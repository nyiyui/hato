package conn

type STState int

const (
	STStateInvalid STState = iota
	STStateBase
	STStateTop
	STStateSide
)

/*
func (s *State) STVal(name STName) (Val, error) {
	s.connsLock.RLock()
	defer s.connsLock.RUnlock()
	s.stsLock.RLock()
	defer s.stsLock.RUnlock()
	connName, ok := s.sts[name]
	if !ok {
		return nil, errors.New("st not found")
	}
	c, ok := s.conns[connName]
	if !ok {
		panic("conn not found (stale st!)")
	}
	if c.GetValue == nil {
		return nil, errors.New("c.GetValue nil")
	}
	return c.GetValue()
}
*/
