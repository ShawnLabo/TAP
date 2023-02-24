// Copyright 2023 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type errorResponse struct {
	Error string `json:"error"`
}

func decodeJSONBody(r *http.Request, v any) error {
	d := json.NewDecoder(r.Body)
	defer r.Body.Close()

	if err := d.Decode(v); err != nil {
		return fmt.Errorf("json.NewDecoder.Decode: %w", err)
	}

	return nil
}

func respondJSON(w http.ResponseWriter, status int, body any) {
	bytes, err := json.Marshal(body)
	if err != nil {
		log.Printf("error: json.Marshal: %s", err.Error())
		http.Error(w, `{"error":"json marshal error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)

	if _, err := w.Write(bytes); err != nil {
		log.Printf("error: w.Write: %s", err.Error())
	}
}

func handleError(w http.ResponseWriter, status int, err error) {
	if status >= 500 {
		log.Printf("error: %s", err)
	}

	e := &errorResponse{
		Error: err.Error(),
	}

	respondJSON(w, status, e)
}
