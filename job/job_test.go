package job

import (
	"time"

	"github.com/stretchr/testify/assert"
	"testing"
)

var testDbPath = ""

func TestScheduleParsing(t *testing.T) {
	cache := NewMockCache()

	fiveMinutesFromNow := time.Now().Add(5 * time.Minute)

	genericMockJob := GetMockJobWithSchedule(2, fiveMinutesFromNow, "P1DT10M10S")

	genericMockJob.Init(cache)

	assert.WithinDuration(
		t, genericMockJob.scheduleTime, fiveMinutesFromNow,
		time.Second, "The difference between parsed time and created time is to great.",
	)
}

func TestBrokenSchedule(t *testing.T) {
	cache := NewMockCache()

	mockJob := GetMockJobWithGenericSchedule()
	mockJob.Schedule = "hfhgasyuweu123"

	err := mockJob.Init(cache)

	assert.Error(t, err)
	assert.Nil(t, mockJob.jobTimer)
}

var delayParsingTests = []struct {
	expected    time.Duration
	intervalStr string
}{
	{time.Hour*24 + time.Second*10 + time.Minute*10, "P1DT10M10S"},
	{time.Second*10 + time.Minute*10, "PT10M10S"},
	{time.Hour*24 + time.Second*10, "P1DT10S"},
	{time.Hour*24*365 + time.Hour*24, "P1Y1DT"},
}

func TestDelayParsing(t *testing.T) {
	testTime := time.Now().Add(time.Minute * 1)

	for _, delayTest := range delayParsingTests {
		cache := NewMockCache()
		genericMockJob := GetMockJobWithSchedule(1, testTime, delayTest.intervalStr)
		genericMockJob.Init(cache)
		assert.Equal(t, delayTest.expected, genericMockJob.delayDuration.ToDuration(), "Parsed duration was incorrect")
	}
}

func TestBrokenDelayHandling(t *testing.T) {
	testTime := time.Now().Add(time.Minute * 1)
	brokenIntervals := []string{
		"DTT",
		"000D",
		"ASDASD",
	}

	for _, intervalTest := range brokenIntervals {
		cache := NewMockCache()

		genericMockJob := GetMockJobWithSchedule(1, testTime, intervalTest)
		err := genericMockJob.Init(cache)

		assert.Error(t, err)
		assert.Nil(t, genericMockJob.jobTimer)
	}
}

func TestJobInit(t *testing.T) {
	cache := NewMockCache()

	genericMockJob := GetMockJobWithGenericSchedule()

	err := genericMockJob.Init(cache)
	assert.Nil(t, err, "err should be nil")

	assert.NotEmpty(t, genericMockJob.Id, "Job.Id should not be empty")
	assert.NotEmpty(t, genericMockJob.jobTimer, "Job.jobTimer should not be empty")
}

func TestJobDisable(t *testing.T) {
	cache := NewMockCache()

	genericMockJob := GetMockJobWithGenericSchedule()
	genericMockJob.Init(cache)

	assert.False(t, genericMockJob.Disabled, "Job should start with disabled == false")

	genericMockJob.Disable()
	assert.True(t, genericMockJob.Disabled, "Job.Disable() should set Job.Disabled to true")
	assert.False(t, genericMockJob.jobTimer.Stop())
}

func TestJobRun(t *testing.T) {
	cache := NewMockCache()

	j := GetMockJobWithGenericSchedule()
	j.Init(cache)
	j.Run(cache)

	now := time.Now()

	assert.Equal(t, j.SuccessCount, uint(1))
	assert.WithinDuration(t, j.LastSuccess, now, 2*time.Second)
	assert.WithinDuration(t, j.LastAttemptedRun, now, 2*time.Second)
}

func TestOneOffJobs(t *testing.T) {
	cache := NewMockCache()

	j := GetMockJob()

	assert.Equal(t, j.SuccessCount, uint(0))
	assert.Equal(t, j.ErrorCount, uint(0))
	assert.Equal(t, j.LastSuccess, time.Time{})
	assert.Equal(t, j.LastError, time.Time{})
	assert.Equal(t, j.LastAttemptedRun, time.Time{})

	j.Init(cache)
	// Find a better way to test a goroutine
	time.Sleep(time.Second)
	now := time.Now()

	assert.Equal(t, j.SuccessCount, uint(1))
	assert.WithinDuration(t, j.LastSuccess, now, 2*time.Second)
	assert.WithinDuration(t, j.LastAttemptedRun, now, 2*time.Second)
	assert.Equal(t, j.scheduleTime, time.Time{})
	assert.Nil(t, j.jobTimer)
}

func TestDependentJobs(t *testing.T) {
	cache := NewMockCache()

	mockJob := GetMockJobWithGenericSchedule()
	mockJob.Name = "mock_parent_job"
	mockJob.Init(cache)

	mockChildJob := GetMockJob()
	mockChildJob.ParentJobs = []string{
		mockJob.Id,
	}
	mockChildJob.Init(cache)

	assert.Equal(t, mockJob.DependentJobs[0], mockChildJob.Id)
	assert.True(t, len(mockJob.DependentJobs) == 1)

	j, err := cache.Get(mockJob.Id)
	assert.NoError(t, err)

	assert.Equal(t, j.DependentJobs[0], mockChildJob.Id)

	j.Run(cache)
	time.Sleep(time.Second * 2)
	n := time.Now()

	assert.WithinDuration(t, mockChildJob.LastAttemptedRun, n, 4*time.Second)
	assert.WithinDuration(t, mockChildJob.LastSuccess, n, 4*time.Second)
}
