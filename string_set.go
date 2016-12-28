package main

import "strings"

type StringSet struct {
	set map[string]struct{}
}

func NewStringSet() StringSet {
	return StringSet{set: map[string]struct{}{}}
}

func (ss *StringSet) Members() []string {
	ms := []string{}
	for m, _ := range ss.set {
		ms = append(ms, m)
	}

	return ms
}

func (ss *StringSet) Set(arg string) error {
	args := strings.Split(arg, ",")
	for _, s := range args {
		if s == "" {
			// split("") results in [""]
			continue
		}

		ss.set[s] = struct{}{}
	}
	return nil
}

func (ss *StringSet) String() string {
	s := ""
	for k := range ss.set {
		s += k + ","
	}

	return strings.TrimRight(s, ",")
}

func (ss StringSet) Get() interface{} {
	return ss.set
}

func (ss *StringSet) Contains(s string) bool {
	_, contains := ss.set[s]
	return contains
}

func (ss *StringSet) IsEmpty() bool {
	return len(ss.set) == 0
}
