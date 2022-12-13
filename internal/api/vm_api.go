// Copyright 2022 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package api

import (
	"bytes"
	"crucible_provider/internal/structs"
	"crucible_provider/internal/util"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// -------------------- API Wrappers --------------------

// CreateVM wraps the the POST function in the VM API that creates a new VM.
//
// param requestBody: The struct representing the VM to be created
//
// param m: map containing provider config info
func CreateVM(requestBody *structs.VMInfo, m map[string]string) error {
	log.Printf("! In create API wrapper")
	auth, err := util.GetAuth(m)
	if err != nil {
		log.Printf("! In create API wrapper, error authenticating")
		return err
	}

	asJSON, err := json.Marshal(requestBody)

	// We encountered an error when encoding the struct as JSON
	if err != nil {
		log.Printf("! In create API wrapper, error encoding request as JSON")
		return err
	}

	// Set up the HTTP request
	req, err := http.NewRequest("POST", util.GetVmApiUrl(m)+"vms", bytes.NewBuffer(asJSON))
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Bearer "+auth)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}

	log.Printf("! JSON being sent to API:\n %v", string(asJSON))
	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("! In create API wrapper, error making HTTP request")
		return err
	}

	log.Printf("! In create API wrapper, request returned with status code %d", resp.StatusCode)
	// Make sure the request succeeded
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("Request returned with status code %d", resp.StatusCode)
	}

	// If we get here, the request was successful
	log.Printf("! Returning from create wrapper without error")
	return nil
}

// GetVMInfo Wraps the VM API call for retrieving vm info.
//
// Param id the id of the VM to look up
//
// Returns a struct containing the VM's info, and a possible error
func GetVMInfo(id string, m map[string]string) (*structs.VMInfo, error) {
	log.Printf("! In read API wrapper")
	// Make the HTTP request
	log.Printf("! In read API wrapper, calling getVMByID helper function")
	resp, err := getVMByID(id, m)
	if err != nil {
		log.Printf("! In read API wrapper, error getting VM")
		return nil, err
	}

	log.Printf("! In read API wrapper, request returned with status code %d", resp.StatusCode)
	// Check if the request was successful
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Request returned with status code %d", resp.StatusCode)
	}

	// Get the VM's info from the response
	ret := unpackResponse(resp)

	// If we're here, no errors have occurred so return data and nil
	log.Printf("! In read API wrapper, returning without error")
	return ret, nil
}

// UpdateVM wraps the for update VM API call.
//
// requestBody: struct representing the data to update the VM with
//
// id: the ID of the VM to be updated
//
// Returns some error on failure and nil on success
func UpdateVM(requestBody *structs.VMInfo, id string, m map[string]string) error {
	log.Printf("! In update API wrapper")
	url := util.GetVmApiUrl(m) + "vms/" + id

	// Get auth token
	auth, err := util.GetAuth(m)
	if err != nil {
		log.Printf("! In update API wrapper, error authenticating")
		return err
	}

	// Encode the request body struct as JSON
	asJSON, err := json.Marshal(requestBody)
	if err != nil {
		log.Printf("! In update API wrapper, error encoding request as JSON")
		return err
	}

	// Set up the request
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(asJSON))
	if err != nil {
		log.Printf("! In update API wrapper, error setting up request")
		return err
	}
	req.Header.Add("Authorization", "Bearer "+auth)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("! In update API wrapper, error making request")
		return err
	}

	log.Printf("! In update API wrapper, request returned with status code %d", resp.StatusCode)
	// Make sure the request succeeded
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Request returned with status code %d", resp.StatusCode)
	}

	log.Printf("! In update API wrapper, returning without error")
	// If we're here, no errors have been encountered so return nil
	return nil
}

// DeleteVM wraps the API to call to delete a given VM.
//
// id: the id of the VM to delete
//
// returns error on failure or nil on success
func DeleteVM(id string, m map[string]string) error {
	log.Printf("! In delete API wrapper")
	url := util.GetVmApiUrl(m) + "vms/" + id

	// Get auth token
	auth, err := util.GetAuth(m)
	if err != nil {
		log.Printf("! In delete API wrapper, error authenticating")
		return err
	}

	// Set up the request
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		log.Printf("! In delete API wrapper, error setting up request")
		return err
	}
	req.Header.Add("Authorization", "Bearer "+auth)
	client := &http.Client{}

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("! In delete API wrapper, error making request")
		return err
	}

	log.Printf("! In delete API wrapper, request returned with status code %d", resp.StatusCode)
	// Check status code
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("Request returned with status code %d", resp.StatusCode)
	}

	log.Printf("! In delete API wrapper, returning without error")
	// If we get here, there were no errors
	return nil
}

// VMExists returns true if a VM with the given id exists
func VMExists(id string, m map[string]string) (bool, error) {
	log.Printf("! In vmExists")
	// Make the HTTP request to get this VM's info
	resp, err := getVMByID(id, m)
	if err != nil {
		log.Printf("! In vmExists, error making http request")
		// The boolean value here will be ignored in the caller since error is non-nil
		return false, err
	}

	log.Printf("! In vmExists, request returned with status code %d, so the function will return %v", resp.StatusCode, resp.StatusCode != http.StatusNotFound)
	return (resp.StatusCode != http.StatusNotFound), nil
}

// RemoveVMFromTeams removes the specified VM from the specified teams
//
// param teams: The IDs of the teams to remove the VM from
//
// param vm: The ID of the VM
//
// param m: A map containing config info for the provider
//
// Returns nil on success or some error on failure
func RemoveVMFromTeams(teams *[]string, vm string, m map[string]string) error {
	auth, err := util.GetAuth(m)
	if err != nil {
		return err
	}

	log.Printf("! In Remove VM from Team API wrapper")

	for _, team := range *teams {
		url := util.GetVmApiUrl(m) + "teams/" + team + "/vms/" + vm
		req, err := http.NewRequest("DELETE", url, nil)
		if err != nil {
			return err
		}
		req.Header.Add("Authorization", "Bearer "+auth)
		client := &http.Client{}

		log.Printf("! url = %v", url)
		log.Printf("! request = %+v", req)

		resp, err := client.Do(req)
		if err != nil {
			return err
		}

		log.Printf("! response: %+v", resp)

		status := resp.StatusCode
		if status != http.StatusNoContent {
			return fmt.Errorf("api returned with status code %d when removing VM %s from team %s", status, vm, team)
		}
	}

	return nil
}

// AddVMToTeams adds the specified VM to the specified teams
//
// param teams: The IDs of the teams to add this VM to
//
// param vm: The ID of the VM
//
// param m: A map containing config info for the provider
//
// Returns nil on success or some error on failure
func AddVMToTeams(teams *[]string, vm string, m map[string]string) error {
	auth, err := util.GetAuth(m)
	if err != nil {
		return err
	}

	log.Printf("! In add team to VM API wrapper")

	for _, team := range *teams {
		url := util.GetVmApiUrl(m) + "teams/" + team + "/vms/" + vm
		req, err := http.NewRequest("POST", url, nil)
		if err != nil {
			return err
		}
		req.Header.Add("Authorization", "Bearer "+auth)
		client := &http.Client{}

		log.Printf("! url = %v", url)
		log.Printf("! request = %+v", req)

		resp, err := client.Do(req)
		if err != nil {
			return err
		}

		log.Printf("! response: %+v", resp)

		status := resp.StatusCode
		if status != http.StatusOK {
			return fmt.Errorf("api returned with status code %d when adding VM %s to team %s", status, vm, team)
		}
	}

	return nil
}

// -------------------- Helper functions --------------------

// Returns the HTTP response from a GET call to get a VM's info
func getVMByID(id string, m map[string]string) (*http.Response, error) {
	log.Printf("! In getVMByID")
	// Get auth token
	auth, err := util.GetAuth(m)
	if err != nil {
		log.Printf("! In getVMByID, error authenticating")
		return nil, err
	}

	// Set up the request
	url := util.GetVmApiUrl(m) + "vms/" + id
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("! In getVMByID, error setting up request")
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+auth)

	// Make the request
	client := &http.Client{}
	resp, err := client.Do(req)

	log.Printf("! response: %+v", resp)

	if err != nil {
		log.Printf("! In getVMByID, error making request")
		return nil, err
	}

	log.Printf("! In getVMByID, returning without error")
	return resp, nil
}

// Unpack the JSON response from a GET call to the API and place it into a vm info struct
func unpackResponse(resp *http.Response) *structs.VMInfo {
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	asStr := buf.String()
	defer resp.Body.Close()

	// Convert the JSON string to a map that we can use to fill our struct
	asMap := make(map[string]interface{})
	json.Unmarshal([]byte(asStr), &asMap)

	log.Printf("! Data returned by GET call:\n%v", asMap)

	teams := asMap["teamIds"].([]interface{})
	teamsConverted := util.ToStringSlice(&teams)

	var connectionPtr *structs.ConsoleConnection
	connection := asMap["consoleConnectionInfo"]
	if connection != nil {
		connectionPtr = structs.ConnectionFromMap(connection.(map[string]interface{}))
	}

	var proxmoxPtr *structs.ProxmoxInfo
	proxmox := asMap["proxmoxVmInfo"]
	if proxmox != nil {
		proxmoxPtr = structs.ProxmoxInfoFromMap(proxmox.(map[string]interface{}))
	}

	// set defaults if defaultUrl and embeddable don't exist (older api versions)
	defaultUrl := false
	defaultUrlObj := asMap["defaultUrl"]

	if defaultUrlObj != nil {
		defaultUrl = defaultUrlObj.(bool)
	}

	embeddable := true
	embeddableObj := asMap["embeddable"]

	if embeddableObj != nil {
		embeddable = embeddableObj.(bool)
	}

	// Unpack the map into a struct. We *should* be able to unmarshal right into the struct, but it's refusing
	// to parse the userId field for some reason. This is logically the same, just rather inelegant
	ret := &structs.VMInfo{
		ID:         asMap["id"].(string),
		URL:        asMap["url"].(string),
		DefaultURL: defaultUrl,
		Name:       asMap["name"].(string),
		TeamIDs:    *teamsConverted,
		UserID:     asMap["userId"],
		Embeddable: embeddable,
		Connection: connectionPtr,
		Proxmox:    proxmoxPtr,
	}
	return ret
}
