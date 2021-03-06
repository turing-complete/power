// Package dynamic provides algorithms for calculating the dynamic power.
package dynamic

import (
	"github.com/ready-steady/sort"
	"github.com/turing-complete/system"
	"github.com/turing-complete/time"
)

// Power is a power calculator.
type Power struct {
	platform    *system.Platform
	application *system.Application
}

// New returns a power calculator.
func New(platform *system.Platform, application *system.Application) *Power {
	return &Power{platform: platform, application: application}
}

// Distribute returns the power consumption of the tasks.
func (self *Power) Distribute(schedule *time.Schedule) []float64 {
	cores, tasks := self.platform.Cores, self.application.Tasks
	power := make([]float64, self.application.Len())
	for i, j := range schedule.Mapping {
		power[i] = cores[j].Power[tasks[i].Type]
	}
	return power
}

// Partition computes a power profile with a variable time step dictated by the
// time moments of power switches.
func (self *Power) Partition(schedule *time.Schedule, ε float64) ([]float64, []float64) {
	return partition(self.Distribute(schedule), schedule, ε)
}

// Sample computes a power profile with respect to a sampling interval Δt.
//
// The required number of samples is specified by ns; short schedules are
// extended while long ones are truncated.
func (self *Power) Sample(schedule *time.Schedule, Δt float64, ns uint) []float64 {
	return sample(self.Distribute(schedule), schedule, Δt, ns)
}

// Progress returns a function for computing the power consumption at an
// arbitrary time moment.
func (self *Power) Progress(schedule *time.Schedule) func(float64, []float64) {
	return progress(self.Distribute(schedule), schedule)
}

func partition(power []float64, schedule *time.Schedule, ε float64) ([]float64, []float64) {
	nc, nt := schedule.Cores, schedule.Tasks

	time := make([]float64, 2*nt)
	copy(time[:nt], schedule.Start)
	copy(time[nt:], schedule.Finish)

	ΔT, steps := traverse(time, ε)
	ssteps, fsteps := steps[:nt], steps[nt:2*nt]

	ns := uint(len(ΔT))

	P := make([]float64, nc*ns)

	for i := uint(0); i < nt; i++ {
		j := schedule.Mapping[i]
		p := power[i]

		s, f := ssteps[i], fsteps[i]

		for ; s < f; s++ {
			P[s*nc+j] = p
		}
	}

	return P, ΔT
}

func progress(power []float64, schedule *time.Schedule) func(float64, []float64) {
	nc, nt := schedule.Cores, schedule.Tasks

	mapping := make([][]uint, nc)
	for i := uint(0); i < nc; i++ {
		mapping[i] = make([]uint, 0, nt)
		for j := uint(0); j < nt; j++ {
			if i == schedule.Mapping[j] {
				mapping[i] = append(mapping[i], j)
			}
		}
	}

	start, finish := schedule.Start, schedule.Finish

	return func(time float64, result []float64) {
		for i := uint(0); i < nc; i++ {
			result[i] = 0
			for _, j := range mapping[i] {
				if start[j] <= time && time <= finish[j] {
					result[i] = power[j]
					break
				}
			}
		}
	}
}

func sample(power []float64, schedule *time.Schedule, Δt float64, ns uint) []float64 {
	nc, nt := schedule.Cores, schedule.Tasks

	P := make([]float64, nc*ns)

	if count := uint(schedule.Span / Δt); count < ns {
		ns = count
	}

	for i := uint(0); i < nt; i++ {
		j := schedule.Mapping[i]
		p := power[i]

		s := uint(schedule.Start[i]/Δt + 0.5)
		f := uint(schedule.Finish[i]/Δt + 0.5)
		if f > ns {
			f = ns
		}

		for ; s < f; s++ {
			P[s*nc+j] = p
		}
	}

	return P
}

func traverse(points []float64, ε float64) ([]float64, []uint) {
	np := uint(len(points))
	order, _ := sort.Quick(points)

	Δ := make([]float64, np-1)
	steps := make([]uint, np)

	j := uint(0)

	for i, x := uint(1), points[0]; i < np; i++ {
		if δ := points[i] - x; δ > ε {
			x = points[i]
			Δ[j] = δ
			j++
		}
		steps[order[i]] = j
	}

	return Δ[:j], steps
}
