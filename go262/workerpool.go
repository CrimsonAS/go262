/*
 * Copyright (c) 2017 Crimson AS <info@crimson.no>
 * Author: Robin Burchell <robin.burchell@crimson.no>
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package go262

import (
	"runtime"
	"sync"
)

// A WorkerPool represents a group of goroutines that are ready and willing to
// do our bidding on some slices of work, called JobQueues.
//
// The idea is that you (persist) a WorkerPool and feed it with JobQueues whenever
// you need something done.
type WorkerPool struct {
	queueChan   chan *JobQueue
	workerCount int
}

// The heart of the worker.
func workerFunc(queueChan chan *JobQueue) {
	// Take a jobqueue...
	for queue := range queueChan {
		// ... and perform all jobs on it, until it is closed. Let the queue
		// know we are alive.
		queue.wg.Add(1)

		for job := range queue.jobchan {
			tr := job.TestCase.Run(job)
			queue.ResultChannel <- tr // ... and send the results back
		}

		// Signal that we are done with the queue.
		queue.wg.Done()
	}
}

// Create a WorkerPool instance. In the process, spawn a bunch of goroutines
// to perform jobs from queues pushed to the pool.
//
// ### we should probably have a pool Cancel method, to close queueChan so
// the workers in a pool return, though we don't need it if all pools are
// persistent...
func NewWorkerPool() *WorkerPool {
	p := &WorkerPool{
		make(chan *JobQueue),
		runtime.NumCPU() - 1,
	}
	for i := 0; i < p.workerCount; i++ {
		go workerFunc(p.queueChan)
	}
	return p
}

// A job queue is a list of things to do
type JobQueue struct {
	// Ask a worker to take care of this job
	jobchan chan *TestJob

	// Workers post results back to this channel
	ResultChannel chan *TestResult

	// Ask the job queue to stop sending jobs
	cancelchan chan int

	// Which pool we're associated with
	pool *WorkerPool

	// Synchronize to make sure all workers in the pool have stopped
	wg sync.WaitGroup
}

// Create a new job queue on a given pool
func NewJobQueue(pool *WorkerPool) *JobQueue {
	q := &JobQueue{
		make(chan *TestJob),
		make(chan *TestResult),
		make(chan int),
		pool,
		sync.WaitGroup{},
	}

	return q
}

// Request that this queue be cancelled. Workers will finish any of their
// current jobs, and then wait for a new queue. The ResultChannel will close
// when workers have acknowledged the cancellation.
func (queue *JobQueue) Cancel() {
	// Stop writing jobs to workers
	queue.cancelchan <- 1
}

// Send a bunch of jobs to the workers in the pool this queue is associated
// with. When the jobs are finished (or the queue is cancelled), the workers
// will be notified to free themselves up for other queued jobs, and the
// ResultChannel is closed (so the caller knows that no more results are
// expected).
//
// ### guard against repeat SendJobs calls, since we close everything when
// cancelled or done? or maybe we should allow multiple?
func (queue *JobQueue) SendJobs(jobs []*TestJob) {
	for i := 0; i < queue.pool.workerCount; i++ {
		queue.pool.queueChan <- queue
	}

	defer func() {
		// Signal we're done...
		close(queue.jobchan)

		// ... and wait for all workers to acknowledge!
		queue.wg.Wait()

		// ... and close the result channel, now workers are done writing to it
		close(queue.ResultChannel)
	}()

	for _, job := range jobs {
		select {
		case <-queue.cancelchan:
			return
		default:
			queue.jobchan <- job
		}
	}
}
