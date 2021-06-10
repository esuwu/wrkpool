package wrkpool

import (
	"bytes"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"runtime"
	"sync"
)

var mutex = &sync.Mutex{}

type Reader interface {
	SetThreadsNumber(n int)
	Read() ([]bytes.Buffer, error)
}

type UrlReader struct {
	urls       []string
	n          int
	threadsSet bool
}

func NewUrlReader(entities ...string) *UrlReader {
	ur := &UrlReader{}
	ur.urls = append(ur.urls, entities...)
	if !ur.threadsSet {
		ur.n = runtime.NumCPU()
	}

	return ur
}

// optional
func (ur *UrlReader) SetThreadsNumber(n int) {
	ur.n = n
	ur.threadsSet = true
}

type BodyResponse struct {
	bytes.Buffer
	err error
}

func (ur *UrlReader) Read() ([]BodyResponse, error) {
	if len(ur.urls) == 0 {
		return nil, errors.New("no urls were sent")
	}
	res := make([]BodyResponse, len(ur.urls))
	taskNumber := len(ur.urls)

	var tasks []*Task
	for i := 0; i < len(ur.urls); i++ {
		task := NewTask(res, ur.urls[i], i)
		tasks = append(tasks, task)
	}
	pool := NewPool(ur.n, &taskNumber, tasks)
	go func() {
		for {
			if taskNumber == 0 {
				pool.Stop()
			}
		}
	}()
	pool.RunBackground()
	return res, nil
}

func (ur *UrlReader) ReadConsistently() ([]bytes.Buffer, error) {
	if len(ur.urls) == 0 {
		return nil, errors.New("no urls were sent")
	}
	res := make([]bytes.Buffer, len(ur.urls))
	for i := 0; i < len(ur.urls); i++ {
		body, err := readBody(ur.urls[i])
		if err != nil {
			return res, err
		}
		res[i].Write(body)
	}
	return res, nil
}

func readBody(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, errors.Errorf("failed to make a request to %s, %v", url, err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Errorf("failed to read body from url %s, %v", url, err)
	}
	return body, nil
}

func process(task *Task, taskCounter *int) {
	resp, err := readBody(task.Url)
	if err != nil {
		mutex.Lock()
		task.Data[task.ID].err = err
		mutex.Unlock()
	}
	if resp != nil {
		mutex.Lock()
		_, err = task.Data[task.ID].Write(resp)
		task.Data[task.ID].err = err
		mutex.Unlock()
	}
	mutex.Lock()
	*taskCounter--
	mutex.Unlock()
}

type Task struct {
	ID   int
	Url  string
	Data []BodyResponse
}

func NewTask(response []BodyResponse, url string, id int) *Task {
	return &Task{Data: response, Url: url, ID: id}
}

type Worker struct {
	taskChan    chan *Task
	quit        chan bool
	taskCounter *int
}

func NewWorker(channel chan *Task, taskCounter *int) *Worker {
	return &Worker{
		taskChan:    channel,
		quit:        make(chan bool),
		taskCounter: taskCounter,
	}
}

func (wr *Worker) StartBackground() {
	for {
		select {
		case task := <-wr.taskChan:
			process(task, wr.taskCounter)
		case <-wr.quit:
			return
		}
	}
}

func (wr *Worker) Stop() {
	go func() {
		wr.quit <- true
	}()
}

type Pool struct {
	Tasks         []*Task
	Workers       []*Worker
	concurrency   int
	collector     chan *Task
	runBackground chan bool
	taskCounter   *int
}

func NewPool(concurrency int, size *int, tasks []*Task) *Pool {
	return &Pool{
		concurrency: concurrency,
		collector:   make(chan *Task, *size),
		taskCounter: size,
		Tasks:       tasks,
	}
}

func (p *Pool) RunBackground() {
	for i := 1; i <= p.concurrency; i++ {
		worker := NewWorker(p.collector, p.taskCounter)
		p.Workers = append(p.Workers, worker)
		go worker.StartBackground()
	}

	for i := range p.Tasks {
		p.collector <- p.Tasks[i]
	}

	p.runBackground = make(chan bool)
	<-p.runBackground
}

func (p *Pool) Stop() {
	for i := range p.Workers {
		p.Workers[i].Stop()
	}
	p.runBackground <- true
}
