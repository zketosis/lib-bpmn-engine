# lib-bpmn-engine: Architecture Documentation

**Table of Contents**

1.  [Introduction](#1-introduction)
2.  [Core Concepts and Design Philosophy](#2-core-concepts-and-design-philosophy)
3.  [Engine Architecture](#3-engine-architecture)
4.  [BPMN Element Handling](#4-bpmn-element-handling)
    *   [4.1. Start Event](#41-start-event)
    *   [4.2. End Event](#42-end-event)
    *   [4.3. Service Task](#43-service-task)
    *   [4.4. User Task](#44-user-task)
    *   [4.5. Gateways](#45-gateways)
        *   [4.5.1. Exclusive Gateway (XOR)](#451-exclusive-gateway-xor)
        *   [4.5.2. Parallel Gateway (AND)](#452-parallel-gateway-and)
        *   [4.5.3. Inclusive Gateway (OR)](#453-inclusive-gateway-or)
        *   [4.5.4. Event-Based Gateway](#454-event-based-gateway)
    *   [4.6. Intermediate Catch Events](#46-intermediate-catch-events)
        *   [4.6.1. Message Intermediate Catch Event](#461-message-intermediate-catch-event)
        *   [4.6.2. Timer Intermediate Catch Event](#462-timer-intermediate-catch-event)
    *   [4.7. Link Intermediate Throw & Catch Event](#47-link-intermediate-throw--catch-event)
    *   [4.8. Sub-Process](#48-sub-process)
5.  [Execution Flow](#5-execution-flow)
6.  [Key Data Structures (Conceptual)](#6-key-data-structures-conceptual)
7.  [Limitations and Considerations](#7-limitations-and-considerations)

## 1. Introduction

`lib-bpmn-engine` is a lightweight, embeddable BPMN 2.0 workflow engine for Go applications. It is designed for in-memory process execution and focuses on simplicity and developer experience. The engine parses BPMN 2.0 XML files and executes the defined workflows. It is not intended to be a full-fledged, standalone BPMN server but rather a library to integrate BPMN capabilities into existing Go services.

## 2. Core Concepts and Design Philosophy

*   **Embeddable:** Designed to be a library, easily integrated into Go applications.
*   **In-Memory:** Primarily operates in-memory. Persistence is supported through marshalling/unmarshalling the engine state to JSON, but database support is intentionally excluded to keep the library lightweight and dependency-free.
*   **Developer Experience:** Aims for a pleasant and straightforward API for developers.
*   **Not Strictly Standard Compliant:** While it supports core BPMN 2.0 elements, it may tolerate some deviations from the strict specification for ease of use. It does not perform extensive BPMN linting or validation.
*   **No Built-in UI or Scheduler:** The engine focuses on the execution core. UI and timer/scheduler functionalities are expected to be handled by the embedding application. It provides event export capabilities to facilitate UI development.
*   **No Locking/Synchronization:** Concurrency control and synchronization are responsibilities of the embedding application.
*   **FEEL for Expressions:** Uses the Friendly Enough Expression Language (FEEL) for evaluating conditions in gateways and other expressions.

## 3. Engine Architecture

The `lib-bpmn-engine` can be conceptualized with the following key components:

*   **BPMN Parser:**
    *   Responsible for reading and parsing BPMN 2.0 XML files.
    *   Transforms the XML definition into an in-memory representation of the process model.
*   **Process Engine Core (`bpmn_engine.New()`):**
    *   The central component that manages process definitions and instances.
    *   Allows loading BPMN process definitions (`LoadFromFile`, `LoadFromBytes`).
    *   Manages the lifecycle of process instances (`CreateAndRunInstance`, `RunOrContinueInstance`).
    *   Handles the execution flow based on the BPMN model.
*   **Process Instance:**
    *   Represents a single execution of a process definition.
    *   Maintains its own state (active, completed, failed) and variables.
    *   Each instance has a unique key.
*   **Task Handlers:**
    *   User-defined Go functions that implement the logic for Service Tasks and User Tasks.
    *   Handlers are registered with the engine and associated with tasks based on:
        *   Task ID (specific to one task)
        *   Task Type (Zeebe's `taskDefinition` extension, for multiple service tasks)
        *   Assignee or Candidate Groups (for User Tasks, Zeebe compatible)
    *   Handlers receive an `ActivatedJob` object, providing access to job details and variables.
    *   Handlers must call `job.Complete()` or `job.Fail()` to signal task completion or failure. Returning without these calls implies an asynchronous/paused task.
*   **Variable Management:**
    *   Process instances and tasks can have associated variables.
    *   Input mappings can create local variables within a task's scope.
    *   Output mappings control how variables from a completed task are merged back into the process instance's scope.
    *   If no output mappings are defined, all task variables are merged.
*   **Event Exporter:**
    *   Provides an interface to export internal engine events, allowing external systems (e.g., UIs, monitoring tools) to observe process execution.
    *   Experimental support for Zeebe Simple Process Monitor.
*   **Timer Management:**
    *   Supports Timer Intermediate Catch Events.
    *   The engine itself does not include a scheduler. When a timer event is encountered, the engine creates a timer event object and pauses the process instance.
    *   The embedding application is responsible for providing an external ticker/scheduler to monitor these scheduled timers and call `RunOrContinueInstance` when a timer is due.
*   **Expression Evaluator (FEEL):**
    *   Integrates a FEEL engine to evaluate expressions, primarily used in gateways (e.g., Exclusive Gateway conditions) and variable mappings.
    *   Supports various data types (numbers, strings, booleans, dates, lists, etc.) and operators (arithmetic, comparison, logical).
*   **Persistence (Marshalling/Unmarshalling):**
    *   Allows the entire engine state (including all process definitions and active instances) to be serialized to JSON (`Marshal()`).
    *   This JSON data can be stored (e.g., file, database) and later deserialized (`Unmarshal()`) to resume paused workflows.
    *   For a large number of instances, it's recommended to use multiple engine instances (e.g., one per process instance) to keep the marshalled data manageable.

## 4. BPMN Element Handling

The engine supports a variety of BPMN 2.0 elements. Here's a callout to how key node types are handled:

### 4.1. Start Event
*   **Handling:** Initiates a new process instance. Multiple start events are supported and triggered in their order of appearance in the BPMN file.

### 4.2. End Event
*   **Handling:** Terminates a path in the process instance. If all active paths reach an end event, the process instance is considered completed. Multiple end events are supported.

### 4.3. Service Task
*   **Function:** Represents an automated task executed by the system.
*   **Handling:**
    *   Requires a registered task handler (Go function).
    *   Handlers can be associated by task ID or task type (using Zeebe's `taskDefinition` extension).
    *   The handler receives an `ActivatedJob` with context (variables, metadata).
    *   The handler executes custom logic and must call `job.Complete()` with optional variables or `job.Fail()` with an error message.
    *   Supports input/output variable mappings.

### 4.4. User Task
*   **Function:** Represents a task that needs to be performed by a human user.
*   **Handling:**
    *   Similar to Service Tasks, requires a registered task handler.
    *   Handlers can be associated by task ID, assignee, or candidate groups (Zeebe compatible).
    *   When a User Task is reached, the engine typically pauses the instance, awaiting external completion (simulated or triggered by a user interaction in the embedding application).
    *   The handler logic would typically involve notifying the user and then, upon user action, calling `job.Complete()` or `job.Fail()`.
    *   If the handler returns without completing/failing, the task remains active (paused state), and `RunOrContinueInstance` needs to be called later to resume.
    *   Supports input/output variable mappings.

### 4.5. Gateways

#### 4.5.1. Exclusive Gateway (XOR)
*   **Function:** Routes the flow to exactly one outgoing sequence flow based on conditions.
*   **Handling:**
    *   Evaluates FEEL expressions defined on outgoing sequence flows.
    *   The first flow whose condition evaluates to `true` is taken.
    *   A default flow (without a condition) can be specified if no other conditions match.
    *   Also used for joining exclusive paths (uncontrolled join).

#### 4.5.2. Parallel Gateway (AND)
*   **Function:** Splits the flow into multiple parallel paths or synchronizes multiple incoming parallel paths.
*   **Handling:**
    *   **Forking:** Activates all outgoing sequence flows concurrently (conceptually; actual execution is sequential in the current design but allows for parallel paths of logic).
    *   **Joining:** Waits for all incoming sequence flows to complete before activating the outgoing flow.

#### 4.5.3. Inclusive Gateway (OR)
*   **Function:** Activates one or more outgoing sequence flows based on conditions. For joining, it waits for all active incoming flows.
*   **Handling:**
    *   Evaluates FEEL expressions on outgoing sequence flows.
    *   All flows whose conditions evaluate to `true` are taken.
    *   A default flow can be specified.

#### 4.5.4. Event-Based Gateway
*   **Function:** Waits for one of several possible events to occur. The first event that occurs determines the path taken.
*   **Handling:** The engine will wait until one of the subsequent events (e.g., Message Catch Event, Timer Catch Event) is triggered. The flow then proceeds along the path of the triggered event.

### 4.6. Intermediate Catch Events

#### 4.6.1. Message Intermediate Catch Event
*   **Function:** Pauses the flow until a specific message is received.
*   **Handling:**
    *   The engine pauses the instance at this event.
    *   An external trigger (e.g., an API call in the embedding application) is needed to publish a message to the engine (`PublishMessage`).
    *   Correlation is currently based on the message name.
    *   Supports output variable mapping from the message payload to the process instance.

#### 4.6.2. Timer Intermediate Catch Event
*   **Function:** Pauses the flow for a specified duration or until a specific date/time.
*   **Handling:**
    *   The engine records a scheduled timer (with a due date/time based on the timer definition, e.g., PT5S for 5 seconds).
    *   The process instance is paused.
    *   An external scheduler (provided by the embedding application) must monitor `bpmnEngine.GetTimersScheduled()`.
    *   When a timer's `DueAt` has passed, the scheduler should call `bpmnEngine.RunOrContinueInstance(instanceKey)` to resume the flow.

### 4.7. Link Intermediate Throw & Catch Event
*   **Function:** Used to create "go-to" connections within the same process level, without a sequence flow line.
*   **Handling:**
    *   When a Link Intermediate Throw Event is reached, the engine looks for a matching Link Intermediate Catch Event (by name) within the same process.
    *   The flow then continues from the corresponding Catch Event.
    *   Supports output variable mapping.

### 4.8. Sub-Process
*   **Function:** Allows for grouping a set of tasks and events, either as an embedded part of the main process or as a reusable (callable) process.
*   **Handling:**
    *   Executes the elements contained within the sub-process.
    *   Can have its own start and end events.
    *   Supports variable input/output mappings to manage data flow between the parent process and the sub-process.

## 5. Execution Flow

1.  **Initialization:** Create a `BpmnEngine` instance (`bpmn_engine.New()`).
2.  **Load Process:** Load a BPMN 2.0 XML file (`engine.LoadFromFile("process.bpmn")`). This parses the XML and stores the process definition.
3.  **Register Handlers:** Register Go functions as handlers for Service Tasks and User Tasks using `engine.NewTaskHandler().Id("task_id").Handler(myHandlerFunc)` or other selectors (type, assignee, candidateGroups).
4.  **Create Instance:** Create a new process instance from a loaded definition: `instance, err := engine.CreateAndRunInstance(process.ProcessKey, initialVariables)`.
    *   This starts execution from the start event(s).
    *   The engine traverses the sequence flows.
5.  **Task Execution:**
    *   When a Service Task or User Task is reached, its registered handler is invoked.
    *   The handler executes its logic.
    *   For synchronous tasks, the handler calls `job.Complete()` or `job.Fail()`.
    *   For asynchronous/paused tasks (common for User Tasks or long-running Service Tasks), the handler might return without completing. The instance remains `Active`.
6.  **Gateway Evaluation:**
    *   Exclusive/Inclusive Gateways: FEEL expressions on outgoing flows are evaluated to determine the path.
    *   Parallel Gateways: Fork or join parallel paths.
7.  **Event Handling:**
    *   Message Catch Events: Instance pauses. `engine.PublishMessage(...)` is needed to resume.
    *   Timer Catch Events: Instance pauses. An external scheduler monitors `engine.GetTimersScheduled()` and calls `engine.RunOrContinueInstance(...)` when due.
8.  **Continuation (for paused instances):**
    *   If an instance was paused (e.g., at a User Task, Timer, or Message Event), `engine.RunOrContinueInstance(instanceKey)` is called to attempt to resume execution. This is typically triggered by an external event (user action, timer due, message received).
9.  **Completion/Failure:**
    *   The instance transitions to `Completed` state when an end event is reached and no other paths are active.
    *   The instance can transition to `Failed` state if a task handler calls `job.Fail()` or an unrecoverable error occurs.
10. **Persistence (Optional):**
    *   At any point, `engine.Marshal()` can be called to get a JSON representation of the engine's state.
    *   `bpmn_engine.Unmarshal(jsonData)` can be used to restore the engine state and resume instances.

## 6. Key Data Structures (Conceptual)

*   `BpmnEngineState`: Holds process definitions, active instances, scheduled timers, registered handlers.
*   `ProcessDefinition`: In-memory representation of a parsed BPMN file (elements, flows, etc.).
*   `ProcessInstanceInfo`: State of a running process (current element, variables, status).
*   `ActivatedJob`: Information passed to task handlers (element ID, process ID, variables, methods to complete/fail).
*   `ScheduledTimer`: Information about a pending timer event (instance key, element ID, due time).

## 7. Limitations and Considerations

*   **Single-threaded Execution Model (within an instance):** While BPMN supports parallelism, the engine processes elements sequentially within a given call to `RunOrContinueInstance`. True parallelism would require the embedding application to manage threads if, for example, multiple service tasks on parallel paths need to execute simultaneously.
*   **No Distributed Timers/Coordination:** The timer mechanism is local to the engine instance. For distributed deployments, a shared, external scheduler and persistence mechanism are crucial.
*   **Error Handling:** Relies on task handlers to manage their errors and use `job.Fail()`. More sophisticated BPMN error events (Error Boundary Events, Error End Events) might have limited or specific handling.
*   **Transactionality:** The engine itself does not provide transactional guarantees across multiple operations or task handlers. This needs to be managed by the embedding application, potentially in conjunction with the persistence mechanism.
*   **Scalability:** Being in-memory, the number of concurrent process instances and the complexity of processes can impact memory usage. The recommendation to use multiple engine instances for large numbers of processes addresses this to some extent for persistence.
