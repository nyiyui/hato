package tal

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/openacid/slimarray/polyfit"
	"github.com/tidwall/buntdb"
	"go.uber.org/zap"
)

type Model2 struct {
	forms     map[uuid.UUID]FormData
	formsLock sync.RWMutex
	dbPath    string
	syncReq   chan struct{}
}

func NewModel2(dbPath string) (*Model2, error) {
	m := &Model2{
		forms:  map[uuid.UUID]FormData{},
		dbPath: dbPath,
	} // 8 was randomly chosen
	err := m.readDB()
	if err != nil {
		return nil, err
	}
	zap.S().Infow("read forms", "forms", m.forms)
	err = m.writeDB()
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (m *Model2) readDB() error {
	m.formsLock.Lock()
	defer m.formsLock.Unlock()
	db, err := buntdb.Open(m.dbPath)
	if err != nil {
		return err
	}
	// TODO: use index "form:*:data"
	err = db.View(func(tx *buntdb.Tx) error {
		err := tx.Ascend("", func(key, value string) bool {
			if !strings.HasPrefix(key, "form:") {
				return true
			}
			if !strings.HasSuffix(key, ":data") {
				return true
			}
			formIRaw := key[5 : len(key)-5]
			formI, err := uuid.Parse(formIRaw)
			if err != nil {
				zap.S().Errorw("parsing key failed",
					"key", key,
					"value", value)
				return true
			}
			var fd FormData
			err = json.Unmarshal([]byte(value), &fd)
			if err != nil {
				zap.S().Errorw("unmarshalling failed",
					"key", key,
					"value", value)
				return true
			}
			m.forms[formI] = fd
			return true
		})
		return err
	})
	return err
}

func (m *Model2) writeDB() error {
	db, err := buntdb.Open(m.dbPath)
	if err != nil {
		return err
	}
	go func() {
		defer db.Close()
		m.syncReq = make(chan struct{}, 8)
		defer panic("writeDB ended")
		for range m.syncReq {
			func() {
				m.formsLock.RLock()
				defer m.formsLock.RUnlock()
				for formI, fd := range m.forms {
					db.Update(func(tx *buntdb.Tx) error {
						fd.UpdateRelation()
						data, err := json.Marshal(fd)
						if err != nil {
							return err
						}
						_, _, err = tx.Set(fmt.Sprintf("form:%s:data", formI), string(data), nil)
						return err
					})
				}
			}()
		}
	}()
	return nil
}

func (m *Model2) RecordTrainCharacter(t *Train) error {
	m.formsLock.Lock()
	defer m.formsLock.Unlock()
	if len(t.History.Spans) == 0 {
		return errors.New("no spans")
	}
	if t.FormI == (uuid.UUID{}) {
		return errors.New("no form specified")
	}
	fd := m.forms[t.FormI]
	added := t.History.Character().Points
	fd.Points = append(fd.Points, added...)
	fd.relationOld = fd.relationOld || len(added) > 0
	m.forms[t.FormI] = fd
	log.Printf("=== fd contains %d points", len(fd.Points))
	m.syncReq <- struct{}{}
	return nil
}

func (m *Model2) GetFormData(formI uuid.UUID) (FormData, bool) {
	fd, ok := m.forms[formI]
	if !ok {
		return FormData{}, false
	}
	return *fd.Clone(), true
}

func (m *Model2) CurrentPosition(t *Train) int64 {
	if t.FormI == (uuid.UUID{}) {
		panic("Train has no FormI")
	}
	fd, ok := m.forms[t.FormI]
	if !ok {
		panic("FormData not found")
	}
	fd.UpdateRelation()
	return t.History.Extrapolate(fd.Relation, time.Now())
}

type FormData struct {
	Points      [][2]int64
	Relation    Relation
	relationOld bool
}

func (fd *FormData) Clone() *FormData {
	points := make([][2]int64, len(fd.Points))
	for i := range fd.Points {
		points[i] = fd.Points[i]
	}
	return &FormData{
		Points: points,
	}
}

type Relation struct {
	Coeffs []float64
	// y = f(x)
	// x : power
	// y : µm/s
}

func (fd *FormData) genRelation() Relation {
	xs := make([]float64, len(fd.Points))
	ys := make([]float64, len(fd.Points))
	for i, point := range fd.Points {
		xs[i] = float64(point[0])
		ys[i] = float64(point[1])
	}
	fit := polyfit.NewFit(xs, ys, 2)
	return Relation{
		Coeffs: fit.Solve(),
	}
}

func (fd *FormData) UpdateRelation() {
	if fd.relationOld {
		fd.Relation = fd.genRelation()
		fd.relationOld = false
	}
}
