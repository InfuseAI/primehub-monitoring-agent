package monitoring

import log "github.com/sirupsen/logrus"

type Metrics struct {
	FifteenMinutes *Buffer
	OneHour        *Buffer
	ThreeHours     *Buffer
	LifeTime       *Buffer
}

func NewMetrics(lifetimeMax int) *Metrics {
	log.Infof("New metrics with lifetime-max %d", lifetimeMax)
	m := new(Metrics)

	// 15m: 10s → 15 * 60 / 10 = 90 points
	m.FifteenMinutes = NewBuffer(10, 90)

	// 1h: 30s → 60 * 60 / 30 = 120 points
	m.OneHour = NewBuffer(30, 120)
	m.OneHour.AverageByLast = m.OneHour.Interval / m.FifteenMinutes.Interval
	log.Debugf("OneHour.AverageByLast=%d", m.OneHour.AverageByLast)

	// 3h: 2m → 3 * 60 * 60 / 120 = 90 points
	m.ThreeHours = NewBuffer(2*60, 90)
	m.ThreeHours.AverageByLast = m.ThreeHours.Interval / m.FifteenMinutes.Interval
	log.Debugf("ThreeHours.AverageByLast=%d", m.ThreeHours.AverageByLast)

	// 4 week: 5m → 4 * 7 * 24 * 60 * 60 / 300 = 8064 points
	m.LifeTime = NewBuffer(5*60, lifetimeMax)
	m.LifeTime.AverageByLast = m.LifeTime.Interval / m.FifteenMinutes.Interval
	log.Debugf("LifeTime.AverageByLast=%d", m.LifeTime.AverageByLast)
	return m
}

func (m *Metrics) Add(record Record) {
	// don't check IsTimeToUpdate here
	// it is controlled by caller, just accept it
	m.FifteenMinutes.Add(record)

	if m.FifteenMinutes.HasLast(m.OneHour.AverageByLast) && m.OneHour.IsTimeToUpdate() {
		m.OneHour.Add(m.FifteenMinutes.LastAverage(m.OneHour.AverageByLast))
	}

	if m.FifteenMinutes.HasLast(m.ThreeHours.AverageByLast) && m.ThreeHours.IsTimeToUpdate() {
		m.ThreeHours.Add(m.FifteenMinutes.LastAverage(m.ThreeHours.AverageByLast))
	}

	if m.FifteenMinutes.HasLast(m.LifeTime.AverageByLast) && m.LifeTime.IsTimeToUpdate() {
		m.LifeTime.Add(m.FifteenMinutes.LastAverage(m.LifeTime.AverageByLast))
	}
}
