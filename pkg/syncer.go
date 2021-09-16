package pkg

import (
	"sync"
)

func SyncZones(config Config) {
	var wg sync.WaitGroup
	for _, job := range config.Jobs {
		if !job.Suspend {
			wg.Add(1)
			go func(job Job) {
				defer wg.Done()
				//@todo: support more cloud providers and dns providers
				eventChannel := getNodeIPChangeEvents(&wg, job)
				for event := range eventChannel {
					syncZone(event, job)
				}
			}(job)
		}
	}
	wg.Wait()
}
