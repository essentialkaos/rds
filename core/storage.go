package core

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"strconv"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// Storage is version agnostic KV data storage
type Storage map[string]string

// ////////////////////////////////////////////////////////////////////////////////// //

// IsEmpty returns true if storage is empty
func (s Storage) IsEmpty() bool {
	return len(s) == 0
}

// Delete removes data for given key
func (s Storage) Delete(key string) {
	delete(s, key)
}

// Has checks if record with given key is exist in storage
func (s Storage) Has(key string) bool {
	_, ok := s[key]

	return ok
}

// Set sets value for given key
func (s Storage) Set(key, value string) {
	s[key] = value
}

// SetI sets integer value for given key
func (s Storage) SetI(key string, value int) {
	s[key] = strconv.Itoa(value)
}

// SetF sets float value for given key
func (s Storage) SetF(key string, value float64) {
	s[key] = strconv.FormatFloat(value, 'f', -1, 64)
}

// SetU sets uint value for given key
func (s Storage) SetU(key string, value uint64) {
	s[key] = strconv.FormatUint(value, 10)
}

// SetB sets boolean value for given key
func (s Storage) SetB(key string, value bool) {
	s[key] = strconv.FormatBool(value)
}

// Get returns value for given key
func (s Storage) Get(key string) string {
	return s[key]
}

// GetI returns integer value for given key
func (s Storage) GetI(key string) (int, error) {
	return strconv.Atoi(s[key])
}

// GetF returns float value for given key
func (s Storage) GetF(key string) (float64, error) {
	return strconv.ParseFloat(s[key], 64)
}

// GetU returns uint value for given key
func (s Storage) GetU(key string) (uint64, error) {
	return strconv.ParseUint(s[key], 10, 64)
}

// GetB returns boolean value for given key
func (s Storage) GetB(key string) (bool, error) {
	return strconv.ParseBool(s[key])
}
