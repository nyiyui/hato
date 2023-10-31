package ctl2

import "sync"

func waitBoth(a, b func()) {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		a()
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		b()
	}()
	wg.Wait()
}
