package main

import (
	"fmt"
	"time"

	"github.com/nitram509/lib-bpmn-engine/pkg/bpmn_engine"
)

// getCurrentNodeInfo returns a string with information about the current state of the process,
// focusing on the element ID if waiting on a timer or message.
func getCurrentNodeInfo(bpmnEngine bpmn_engine.BpmnEngineState, instanceKey int64) string {
	// Get basic instance information
	instance := bpmnEngine.FindProcessInstance(instanceKey)
	if instance == nil {
		return fmt.Sprintf("instance-%d (not found)", instanceKey)
	}

	// Default info is the instance key
	info := fmt.Sprintf("instance-%d", instanceKey)

	// Check for timers
	timers := bpmnEngine.GetTimersScheduled()
	for _, timer := range timers {
		if timer.ProcessInstanceKey == instanceKey {
			// Return the ElementId of the timer event
			return fmt.Sprintf("%s (Waiting at Timer: %s, due in %v)",
				info,
				timer.ElementId, // Use the ElementId here
				time.Until(timer.DueAt).Round(time.Second))
		}
	}

	// Check for message subscriptions
	subs := bpmnEngine.GetMessageSubscriptions()
	for _, sub := range subs {
		if sub.ProcessInstanceKey == instanceKey {
			// Return the ElementId of the message event
			return fmt.Sprintf("%s (Waiting at Message Event: %s for message '%s')",
				info,
				sub.ElementId, // Use the ElementId here
				sub.Name)      // Corrected field name from MessageName to Name
		}
	}

	// If not waiting on a specific timer or message event known to the engine state,
	// return the basic instance info. A more sophisticated approach might involve
	// inspecting the instance's internal state if the API allowed.
	return info
}

func main() {
	bpmnEngine := bpmn_engine.New()

	// BPMN file names to load
	bpmnFiles := []string{
		"timeout-example1.bpmn",
		"timeout-example2.bpmn",
		"timeout-example3.bpmn",
	}

	// Just some dummy handler to complete the tasks/jobs
	registerDummyTaskHandlers(&bpmnEngine)

	// Load three different BPMN files and start one process instance from each
	instanceKeys := []int64{}
	for _, file := range bpmnFiles {
		process, err := bpmnEngine.LoadFromFile(file)
		if err != nil {
			panic(fmt.Sprintf("file %q can't be read.", file))
		}
		instance, err := bpmnEngine.CreateAndRunInstance(process.ProcessKey, nil)
		if err != nil {
			panic(fmt.Sprintf("failed to create and run process instance from file %q", file))
		}
		instanceKeys = append(instanceKeys, instance.GetInstanceKey())
		println(fmt.Sprintf("Process Instance from file %q ID: %d", file, instance.GetInstanceKey()))
	}

	// Monitor all instances
	for len(instanceKeys) > 0 {
		activeInstances := []int64{}
		for _, instanceKey := range instanceKeys {
			instance := bpmnEngine.FindProcessInstance(instanceKey)
			if instance == nil {
				// Instance might have completed between checks
				continue
			}
			state := instance.GetState()
			if state == bpmn_engine.Active {
				// Get the current node info, which now includes the element ID if waiting
				nodeInfo := getCurrentNodeInfo(bpmnEngine, instanceKey)
				println(fmt.Sprintf("tick. %s", nodeInfo)) // Print the detailed node info
				_, err := bpmnEngine.RunOrContinueInstance(instanceKey)
				if err != nil {
					println(fmt.Sprintf("Error continuing instance %d: %v", instanceKey, err))
					// Decide how to handle errors, e.g., remove the instance from monitoring
				} else {
					// Check state again after attempting to continue
					updatedInstance := bpmnEngine.FindProcessInstance(instanceKey)
					if updatedInstance != nil && updatedInstance.GetState() == bpmn_engine.Active {
						activeInstances = append(activeInstances, instanceKey)
					} else if updatedInstance != nil {
						println(fmt.Sprintf("Process Instance ID %d changed state to: %s", instanceKey, updatedInstance.GetState()))
					} else {
						println(fmt.Sprintf("Process Instance ID %d likely completed.", instanceKey))
					}
				}
			} else {
				// This case might be reached if the instance completed before the RunOrContinueInstance call
				println(fmt.Sprintf("Process Instance ID %d already completed with state: %s", instanceKey, state))
			}
		}
		instanceKeys = activeInstances
		if len(instanceKeys) > 0 {
			time.Sleep(2 * time.Second)
		}
	}
	println("All process instances have completed.")
}
