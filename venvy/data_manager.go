package venvy

import (
	"encoding/json"
	"fmt"
	"github.com/peterbourgon/diskv"
	logger "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"strings"
)

type DataManager struct {
	storageDir string
	diskKV     *diskv.Diskv
}

func (dm *DataManager) setup() error {
	dataDir := filepath.Join(dm.storageDir, "kvData")
	err := os.MkdirAll(dataDir, 0700) // also creates storageDir
	if err != nil {
		return err
	}
	splitColonTransform := func(s string) []string { return strings.Split(s, ":") }
	dm.diskKV = diskv.New(diskv.Options{
		BasePath:     dataDir,
		Transform:    splitColonTransform,
		CacheSizeMax: 1024 * 1024,
	})
	return nil
}

func (dm *DataManager) ChDir(toDir string) error {
	dm.storageDir = toDir
	return dm.setup()
}

func (dm *DataManager) Reset() error {
	err := dm.diskKV.EraseAll()
	if err != nil {
		return err
	}
	err = os.RemoveAll(dm.storageDir)
	if err != nil {
		return err
	}
	return dm.setup()
}

func (dm *DataManager) StoragePath(elem ...string) string {
	return filepath.Join(append([]string{dm.storageDir}, elem...)...)
}

func (dm *DataManager) SetKey(key string, value string) error {
	return dm.diskKV.Write(key, []byte(value))
}

func (dm *DataManager) GetKey(key string) (string, error) {
	data, err := dm.diskKV.Read(key)
	if err != nil {
		return "", err
	} else {
		return string(data), nil
	}
}

func (dm *DataManager) WriteJson(key string, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return dm.diskKV.Write(key, data)
}

func (dm *DataManager) ReadJson(key string, v interface{}) error {
	data, err := dm.diskKV.Read(key)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func NewDataManager(storageDir string) (*DataManager, error) {
	logger.Debugf("Creating data store at %s", storageDir)
	if !filepath.IsAbs(storageDir) {
		return nil, fmt.Errorf("storage dir %s not an absolute path", storageDir)
	}
	dm := &DataManager{storageDir: storageDir}
	return dm, dm.setup()
}
