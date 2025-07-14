package task

import (
	"crypto/rand"
	"fmt"
	"main/internal/config"
	"main/internal/filework"
	"math/big"
	"sync"
)

type Task struct {
	Mutex *sync.Mutex
	List  map[int]filework.Zip
}

func (t *Task) Add(max int, env config.Config) (int, error) {
	if len(t.List) >= max {
		return 0, fmt.Errorf("server is busy")
	}
	t.Mutex.Lock()
	defer t.Mutex.Unlock()
	id, err := t.getId()
	if err != nil {
		return 0, err
	}
	t.List[id] = *filework.PrepareZip(env)
	return id, nil
}

func (t *Task) Drop(id int) {
	t.Mutex.Lock()
	defer t.Mutex.Unlock()
	delete(t.List, id)
}

func (t *Task) getId() (int, error) {
	for range 10 {
		id, err := rand.Int(rand.Reader, big.NewInt(10000000000))
		if err != nil {
			return 0, err
		}
		_, ok := t.List[int(id.Int64())]
		if ok {
			continue
		}
		return int(id.Int64()), nil
	}
	return 0, fmt.Errorf("failed create id")
}
