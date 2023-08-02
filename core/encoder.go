package core

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2023 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"encoding/json"
	"fmt"
	"os"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// encodeAndSave encodes given struct to JSON and writes it to file
func encodeAndSave(data interface{}, path string) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")

	if err != nil {
		return fmt.Errorf("Can't encode data: %v", err)
	}

	jsonData = append(jsonData, '\n')

	fd, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0640)

	if err != nil {
		return fmt.Errorf("Can't open file for writing: %v", err)
	}

	defer fd.Close()

	_, err = fd.Write(jsonData)

	if err != nil {
		return fmt.Errorf("Can't save data: %v", err)
	}

	return nil
}

// readAndDecode reads data from files and decodes JSON encoded data
func readAndDecode(data interface{}, path string) error {
	rawData, err := os.ReadFile(path)

	if err != nil {
		return fmt.Errorf("Can't read file: %v", err)
	}

	err = json.Unmarshal(rawData, data)

	if err != nil {
		return fmt.Errorf("Can't decode data: %v", err)
	}

	return nil
}
