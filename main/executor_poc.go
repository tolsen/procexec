package main

import (
	"github.com/louisaberger/procexec"

	"fmt"
	"sync"
	"time"
)

type MyExecutor struct {
	stopChan  chan struct{}
	processWG *sync.WaitGroup // all spawned processes
}

func NewMyExecutor() *MyExecutor {
	return &MyExecutor{}
}

func (self *MyExecutor) Execute(settings map[string]interface{}, panicChan chan *procexec.GoroutinePanic) error {

	// set up Process Wait Group
	var wg sync.WaitGroup
	self.processWG = &wg

	// Set up stop chan
	self.stopChan = make(chan struct{}, 1)

	// Do some setup before starting the agent
	if err := executeSetup(); err != nil {
		return err
	}

	// Some other setup for the agent is spawned off in a separate goroutine
	procexec.PanicCapturingGo(func() { nestedFunctionToSpawn(self.stopChan) }, panicChan, self.processWG)

	// Start our main function that will be running asynchronously
	procexec.PanicCapturingGo(func() { RunAgent(self.processWG, self.stopChan, panicChan) }, panicChan, self.processWG)
	return nil
}

func executeSetup() error {
	if false {
		return fmt.Errorf("Some error setting up for execute")
	}
	return nil
}

func (self *MyExecutor) Stop() error {
	close(self.stopChan)

	return self.waitForShutdown()
}

func (self *MyExecutor) waitForShutdown() error {
	s := make(chan struct{})
	t := time.NewTimer(2 * time.Minute)

	go func() {
		self.processWG.Wait()
		close(s)
	}()

	select {
	case <-s:
		return nil
	case <-t.C:
		return fmt.Errorf("Timed out waiting for shut down")
	}
}

func RunAgent(processWG *sync.WaitGroup, stopChan chan struct{}, panicChan chan *procexec.GoroutinePanic) {
	// Example of a nested spawned go routine in RunAgent
	procexec.PanicCapturingGo(func() { nestedFunctionToSpawn(stopChan) }, panicChan, processWG)

	i := 0
	for {
		select {
		case <-stopChan:
			return
		default:
			// do nothing / prevent blocking
		}

		fmt.Printf("Running iteration %v...\n", i)
		i++

		time.Sleep(time.Second)
	}
}

func nestedFunctionToSpawn(stopChan chan struct{}) {
	for {
		select {
		case <-stopChan:
			return
		default:
		}
		time.Sleep(5 * time.Second)
	}
}

var _ procexec.Executor = (*MyExecutor)(nil)

func main() {
	var pe procexec.Executor = NewMyExecutor()
	panicChan := make(chan *procexec.GoroutinePanic, 128)

	settings := map[string]interface{}{}

	if err := pe.Execute(settings, panicChan); err != nil {
		panic(fmt.Sprintf("Error starting : %v", err))
	}

	if err := pe.Stop(); err != nil {
		panic(fmt.Sprintf("Error stopping : %v", err))
	}

	if err := pe.Execute(settings, panicChan); err != nil {
		panic(fmt.Sprintf("Error starting : %v", err))
	}

	if err := pe.Stop(); err != nil {
		panic(fmt.Sprintf("Error stopping : %v", err))
	}

}
