package store

import (
	"netman-agent/models"
	"sync"
)

type Store struct {
	data map[string][]models.Container
	lock *sync.Mutex
}

func New() *Store {
	return &Store{
		data: make(map[string][]models.Container),
		lock: new(sync.Mutex),
	}
}

func (s *Store) GetContainers() map[string][]models.Container {
	s.lock.Lock()
	defer s.lock.Unlock()
	toReturn := make(map[string][]models.Container)
	for k, v := range s.data {
		toReturn[k] = v
	}
	return toReturn
}

func (s *Store) Add(containerID, groupID, IP string) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.data[groupID] = append(s.data[groupID], models.Container{ID: containerID, IP: IP})
	return nil
}

func (s *Store) Del(containerID string) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	for container_group, containers := range s.data {
		for i, container := range containers {
			if container.ID == containerID {
				shorter := removeFromList(containers, i)
				if len(shorter) > 0 {
					s.data[container_group] = shorter
				} else {
					delete(s.data, container_group)
				}
				return nil
			}
		}
	}
	return nil
}

func removeFromList(list []models.Container, i int) []models.Container {
	return append(list[:i], list[i+1:]...)
}
