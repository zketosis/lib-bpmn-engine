package main

import (
	"fmt"
	"time"

	"github.com/nitram509/lib-bpmn-engine/pkg/bpmn_engine"
)

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
			// Retrieve the process instance using FindProcessInstance
			instance := bpmnEngine.FindProcessInstance(instanceKey)
			if instance == nil {
				println(fmt.Sprintf("Error retrieving instance %d: instance not found", instanceKey))
				continue
			}
			state := instance.GetState()
			if state == bpmn_engine.Active {
				println(fmt.Sprintf("tick. Process Instance ID: %d", instanceKey))
				_, err := bpmnEngine.RunOrContinueInstance(instanceKey)
				if err != nil {
					println(fmt.Sprintf("Error continuing instance %d: %v", instanceKey, err))
				} else {
					activeInstances = append(activeInstances, instanceKey)
				}
			} else {
				println(fmt.Sprintf("Process Instance ID %d completed with state: %s", instanceKey, state))
			}
		}
		instanceKeys = activeInstances
		time.Sleep(2 * time.Second)
	}
}
