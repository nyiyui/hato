package tal

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/openacid/slimarray/polyfit"
	"github.com/tidwall/buntdb"
	"go.uber.org/zap"
	"nyiyui.ca/hato/sakayukari/tal/layout"
)

type Model2 struct {
	g            *Guide
	forms        map[uuid.UUID]FormData
	formsLock    sync.RWMutex
	dbPath       string
	syncReq      chan struct{}
	ignoreWrites bool
}

func NewModel2(g *Guide, dbPath string) (*Model2, error) {
	m := &Model2{
		g:      g,
		forms:  map[uuid.UUID]FormData{},
		dbPath: dbPath,
	}
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
	if m.ignoreWrites {
		zap.S().Debugf("model2: ignored write")
		return nil
	}
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
	zap.S().Warnf("RecordTrainCharacter disabled for now!")
	return nil
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
	fd.latestRelation = fd.latestRelation && len(added) == 0
	m.forms[t.FormI] = fd
	log.Printf("=== fd contains %d points", len(fd.Points))
	m.syncReq <- struct{}{}
	return nil
}

func (m *Model2) GetFormData(formI uuid.UUID) (FormData, bool) {
	m.formsLock.Lock()
	defer m.formsLock.Unlock()
	fd, ok := m.forms[formI]
	if !ok {
		return FormData{}, false
	}
	return *fd.Clone(), true
}

//func (m *Model2) SetPosition(t *Train, pos layout.Position) {
//	m.formsLock.Lock()
//	defer m.formsLock.Unlock()
//	if t.FormI == (uuid.UUID{}) {
//		panic("Train has no FormI")
//	}
//	m.formsLock.Lock()
//	defer m.formsLock.Unlock()
//	fd, ok := m.forms[t.FormI]
//	if !ok {
//		panic("FormData not found")
//	}
//	fd.latestPosition = pos
//	m.forms[t.FormI] = fd
//}

// CurrentPosition2 returns CurrentPosition without overrun, so it can be used with text/template.
func (m *Model2) CurrentPosition2(t *Train) layout.Position {
	pos, _ := m.CurrentPosition(t)
	return pos
}

func (m *Model2) CurrentPosition3(t *Train, fence bool) (pos layout.Position, overrun bool) {
	offset := m.CurrentOffset(t)
	pos, err := m.g.Layout.OffsetToPosition(*t.Path, offset)
	if err != nil {
		// fallback to end of path
		pos = m.g.Layout.LinePortToPosition(t.Path.Follows[len(t.Path.Follows)-1])
		overrun = true
	}
	if fence {
		c := GuideFence(m.g.Layout, t)
		//zap.S().Debugf("pos nofit = %#v", pos)
		pos = FitInConstraint(m.g.Layout, c, pos)
		//zap.S().Debugf("pos fit = %#v", pos)
	}
	return
}

func (m *Model2) CurrentPosition(t *Train) (pos layout.Position, overrun bool) {
	return m.CurrentPosition3(t, true)
}

// CurrentOffset returns the estimated offset. Note that it doesn't account for any constraints.
func (m *Model2) CurrentOffset(t *Train) int64 {
	if t.FormI == (uuid.UUID{}) {
		panic("Train has no FormI")
	}
	m.formsLock.Lock()
	defer m.formsLock.Unlock()
	fd, ok := m.forms[t.FormI]
	if !ok {
		panic("FormData not found")
	}
	fd.UpdateRelation()
	m.forms[t.FormI] = fd
	offset, ok := t.History.Extrapolate(m.g.Layout, *t.Path, fd.Relation, time.Now())
	if !ok {
		return -1
	}
	return offset
}

func (m *Model2) SetIgnoreWrites() {
	m.ignoreWrites = true
}

type FormData struct {
	Points         [][2]int64
	Relation       Relation
	latestRelation bool

	//latestPosition layout.Position
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

// SolveForX solves for x from y in the relation.
// For quadratic equations: if two distinct solutions are found, choose one that is in range (0-255 inclusive). If both are in range, choose the lower value (this choice is completely arbitrary).
func (r Relation) SolveForX(y float64) (x float64, ok bool) {
	const min = 0
	const max = 255
	switch len(r.Coeffs) {
	case 0:
		panic("cannot solve for literally nothing")
	case 1:
		panic("cannot solve for constant")
	case 2:
		// x=(y-a)/b
		x := (y - r.Coeffs[0]) / r.Coeffs[1]
		return x, x >= min && x <= max
	case 3:
		// y=ax^2+bx+c
		// 0=ax^2+bx+c-y
		//   -b±sqrt(b^2-4ac)
		// x=----------------
		//   2a
		a := r.Coeffs[2]
		b := r.Coeffs[1]
		c := r.Coeffs[0] - y
		xa := (-b + math.Sqrt(b*b-4*a*c)) / (2 * a)
		xb := (-b - math.Sqrt(b*b-4*a*c)) / (2 * a)
		xaInRange := xa >= min && xa <= max
		xbInRange := xb >= min && xb <= max
		if xaInRange && !xbInRange {
			return xa, true
		} else if !xaInRange && xbInRange {
			return xb, true
		} else if !xaInRange && !xbInRange {
			return 0, false
		} else if xaInRange && xbInRange {
			return math.Min(xa, xb), true
		} else {
			panic("unreachable")
		}
	default:
		panic(fmt.Sprintf("only linear and quadratic equations supported (%d coeffs given)", len(r.Coeffs)))
	}
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
	if !fd.latestRelation {
		fd.Relation = fd.genRelation()
		fd.latestRelation = true
	}
}
