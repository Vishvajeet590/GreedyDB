package db

import (
	"container/heap"
	"fmt"
	"sync"
	"time"
)

const (
	SetIfKeyNotExist = iota
	SetIfKeyExist
)

var pq = PriorityQueue{}

// DataBlock Used to store Value in map
type DataBlock struct {
	value string
}

type Datastore struct {
	mu   sync.RWMutex
	Data map[string]DataBlock
	List map[string][]string
}

// Store Interface for Datastore
type Store interface {
	Set(query *ParsedQuery) error
	Get(query *ParsedQuery) (string, error)
	QPush(query *ParsedQuery) error
	QPop(query *ParsedQuery) (string, error)
	BQPop(query *ParsedQuery) (interface{}, error)
}

func NewDataStore() *Datastore {
	pq = make(PriorityQueue, 0)
	heap.Init(&pq)
	store := Datastore{
		Data: make(map[string]DataBlock),
		List: make(map[string][]string),
	}

	//Stating a goroutine to check if the top item of PQ has expired or not.
	go store.ActiveExpiry()

	return &store
}

func (ds *Datastore) Set(query *ParsedQuery) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	valueBlock := DataBlock{
		value: query.Value,
	}

	switch query.KeyExistCondition {
	case SetIfKeyNotExist:
		if _, ok := ds.Data[query.Key]; !ok {
			ds.Data[query.Key] = valueBlock
		}
	case SetIfKeyExist:
		if _, ok := ds.Data[query.Key]; ok {
			ds.Data[query.Key] = valueBlock
		} else {
			return fmt.Errorf("key does not exist hence not SET")
		}
	default:
		ds.Data[query.Key] = valueBlock

	}

	//Adding element in PQ for Active expiry after X seconds.
	if query.Expiry {
		heap.Push(&pq, &Item{
			KeyName: query.Key,
			Expiry:  time.Now().Add(query.ExpiryTime),
		})
	}

	return nil
}

func (ds *Datastore) Get(query *ParsedQuery) (string, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	data, ok := ds.Data[query.Key]
	if !ok {
		return "", fmt.Errorf("key does not exist or expired")
	}

	return data.value, nil
}

func (ds *Datastore) QPush(query *ParsedQuery) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if len(query.ListValues) == 0 {
		return fmt.Errorf("list require atleast 1 value")
	}

	//checking if list exist
	if _, ok := ds.List[query.ListName]; !ok {
		ds.List[query.Key] = make([]string, 0)
	}

	//Adding list values to the mapped list
	ds.List[query.ListName] = append(ds.List[query.ListName], query.ListValues...)
	return nil
}

func (ds *Datastore) QPop(query *ParsedQuery) (string, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	listItem, ok := ds.List[query.ListName]
	if !ok {
		return "", fmt.Errorf("key does not exist")
	}
	if len(listItem) == 0 {
		return "", fmt.Errorf("list is empty")
	}
	poppedValue := listItem[len(listItem)-1]
	listItem = listItem[:len(listItem)-1]
	ds.List[query.ListName] = listItem
	return poppedValue, nil
}

func (ds *Datastore) BQPop(query *ParsedQuery) (interface{}, error) {
	listItem, ok := ds.List[query.ListName]
	if !ok {
		return "", fmt.Errorf("key does not exist")
	}
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if len(listItem) == 0 {
		if query.ExpiryTime == 0 {
			return "", nil
		}

		timer := time.NewTimer(query.ExpiryTime)

		for {
			ds.mu.Unlock()
			time.Sleep(1 * time.Second)
			ds.mu.Lock()
			select {
			case <-timer.C:
				return nil, nil
			default:
			}
			if len(ds.List[query.ListName]) > 0 {
				break
			}
		}
		if !timer.Stop() {
			<-timer.C
		}
	}

	poppedValue := ds.List[query.ListName][len(ds.List[query.ListName])-1]
	ds.List[query.ListName] = ds.List[query.ListName][:len(ds.List[query.ListName])-1]
	return poppedValue, nil
}

func (ds *Datastore) ActiveExpiry() {
	forever := make(chan bool)
	go func() {
		for {
			time.Sleep(1 * time.Second)
			item, err := pq.PeekTopPriority()
			if err != nil {
				//PQ could be empty
				continue
			}

			//Item Expired so deleting from memory
			if !time.Now().Before(item.Expiry) {
				ds.mu.Lock()
				delete(ds.Data, item.KeyName)
				heap.Pop(&pq)
				ds.mu.Unlock()
			}
		}
	}()
	<-forever
}
