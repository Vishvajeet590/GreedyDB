package db

import (
	"fmt"
	"sync"
	"time"
)

type DataBlock struct {
	value   string
	expiry  time.Time
	persist bool
}

type Datastore struct {
	mu   sync.RWMutex
	Data map[string]DataBlock
	List map[string][]string
}

const (
	SetIfKeyNotExist = iota
	SetIfKeyExist    = iota
)

type Store interface {
	Set(query *Query) error
	Get(key string) (string, error)
	QPush(query *Query) error
	QPop(query *Query) (string, error)
}

func NewDataStore() *Datastore {
	return &Datastore{
		Data: make(map[string]DataBlock),
		List: make(map[string][]string),
	}
}

func (ds *Datastore) Set(query *Query) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	valueBlock := DataBlock{
		value: query.Value,
	}

	if query.Expiry {
		valueBlock.expiry = time.Now().Add(query.ExpiryTime)
		valueBlock.persist = false
	} else {
		valueBlock.persist = true
	}

	switch query.KeyExistCondition {
	case SetIfKeyNotExist:
		if _, ok := ds.Data[query.Key]; !ok {
			ds.Data[query.Key] = valueBlock
		}
	case SetIfKeyExist:
		if _, ok := ds.Data[query.Key]; ok {
			ds.Data[query.Key] = valueBlock
		}
	default:
		ds.Data[query.Key] = valueBlock

	}
	return nil
}

func (ds *Datastore) Get(key string) (string, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	data, ok := ds.Data[key]
	if !ok {
		return "", fmt.Errorf("key does not exist")
	}

	if ok && data.persist {
		return data.value, nil
	} else if ok && time.Now().Before(data.expiry) {
		return data.value, nil
	}
	return "", fmt.Errorf("expired")
}

func (ds *Datastore) QPush(query *Query) error {
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

func (ds *Datastore) QPop(query *Query) (string, error) {
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
