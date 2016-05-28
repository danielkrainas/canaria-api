package memory

import (
	"errors"
	"sync"

	"github.com/danielkrainas/canaria-api/common"
	"github.com/danielkrainas/canaria-api/context"
	"github.com/danielkrainas/canaria-api/storage"
	"github.com/danielkrainas/canaria-api/storage/factory"
)

const driverName = "memory"

func init() {
	factory.Register(driverName, &memoryDriverFactory{})
}

type memoryDriverFactory struct{}

func (factory *memoryDriverFactory) Create(parameters map[string]interface{}) (storage.StorageDriver, error) {
	return New(), nil
}

type driver struct {
	hooks    *hookStorage
	canaries *canaryStorage
}

func New() *driver {
	return &driver{
		canaries: &canaryStorage{
			deleted:  make(map[string]common.Canary),
			canaries: make(map[string]common.Canary),
		},
		hooks: &hookStorage{
			hooks:         make(map[string]common.WebHook),
			hooksByCanary: make(map[string][]common.WebHook),
		},
	}
}

func (d *driver) Hooks() storage.HookStorage {
	return d.hooks
}

func (d *driver) Canaries() storage.CanaryStorage {
	return d.canaries
}

type hookStorage struct {
	mu            sync.Mutex
	hooks         map[string]common.WebHook
	hooksByCanary map[string][]common.WebHook
}

func (hs *hookStorage) Get(ctx context.Context, id string) (*common.WebHook, error) {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	wh, ok := hs.hooks[id]
	if !ok {
		return nil, errors.New("entry not found")
	}

	return &wh, nil
}

func (hs *hookStorage) Store(ctx context.Context, wh *common.WebHook) error {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	hs.hooks[wh.ID] = *wh
	hooks, ok := hs.hooksByCanary[wh.CanaryID]
	if !ok {
		hooks = make([]common.WebHook, 1)
		hs.hooksByCanary[wh.CanaryID] = hooks
	}

	hs.hooksByCanary[wh.CanaryID] = append(hooks, *wh)
	return nil
}

func (hs *hookStorage) Delete(ctx context.Context, id string) error {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	wh, ok := hs.hooks[id]
	if !ok {
		return nil
	}

	delete(hs.hooks, id)
	hooks, ok := hs.hooksByCanary[wh.CanaryID]
	if ok {
		for i := 0; i < len(hooks); i++ {
			if hooks[i].ID == id {
				hooks = append(hooks[:i], hooks[i+1:]...)
				break
			}
		}

		hs.hooksByCanary[wh.CanaryID] = hooks
	}

	return nil
}

func (hs *hookStorage) GetForCanary(ctx context.Context, canaryID string) ([]*common.WebHook, error) {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	hooks := make([]*common.WebHook, 0)
	vhooks, ok := hs.hooksByCanary[canaryID]
	if !ok {
		return hooks, nil
	}

	for _, wh := range vhooks {
		hooks = append(hooks, &wh)
	}

	return hooks, nil
}

func (hs *hookStorage) DeleteForCanary(ctx context.Context, canaryID string) ([]string, error) {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	hooks, ok := hs.hooksByCanary[canaryID]
	if !ok {
		return []string{}, nil
	}

	ids := make([]string, len(hooks))
	for _, wh := range hooks {
		ids = append(ids, wh.ID)
		delete(hs.hooks, wh.ID)
	}

	delete(hs.hooksByCanary, canaryID)
	return ids, nil
}

type canaryStorage struct {
	mu       sync.Mutex
	canaries map[string]common.Canary
	deleted  map[string]common.Canary
}

func (cs *canaryStorage) IsDeleted(ctx context.Context, id string) bool {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	_, found := cs.deleted[id]
	return found
}

func (cs *canaryStorage) Get(ctx context.Context, id string) (*common.Canary, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	c, ok := cs.canaries[id]
	if !ok {
		return nil, errors.New("entry not found")
	}

	return &c, nil
}

func (cs *canaryStorage) Store(ctx context.Context, c *common.Canary) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.canaries[c.ID] = *c
	return nil
}

func (cs *canaryStorage) Delete(ctx context.Context, id string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if c, ok := cs.canaries[id]; ok {
		delete(cs.canaries, id)
		cs.deleted[id] = c
	}

	return nil
}
