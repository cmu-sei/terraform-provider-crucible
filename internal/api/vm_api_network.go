// Copyright 2022 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/cmu-sei/terraform-provider-crucible/internal/structs"
	"github.com/cmu-sei/terraform-provider-crucible/internal/util"
	"log"
	"net/http"
)

// CreateViewNetwork wraps the POST call to create a view network in the VM API.
func CreateViewNetwork(network *structs.ViewNetworkInfo, m map[string]string) (*structs.ViewNetworkInfo, error) {
	log.Printf("! In CreateViewNetwork API wrapper")

	auth, err := util.GetAuth(m)
	if err != nil {
		return nil, err
	}

	payload := map[string]interface{}{
		"providerType":       network.ProviderType,
		"providerInstanceId": network.ProviderInstanceId,
		"networkId":          network.NetworkId,
		"name":               network.Name,
		"teamIds":            network.TeamIds,
	}

	asJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	url := util.GetVmApiUrl(m) + "views/" + network.ViewID + "/networks"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(asJSON))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+auth)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("VM API returned status %d when creating view network for view %s", resp.StatusCode, network.ViewID)
	}

	result, err := unpackViewNetworkResponse(resp)
	if err != nil {
		return nil, err
	}

	log.Printf("! ViewNetwork created with ID %s", result.ID)
	return result, nil
}

// GetViewNetwork wraps the GET call to read a single view network.
func GetViewNetwork(viewID, id string, m map[string]string) (*structs.ViewNetworkInfo, error) {
	log.Printf("! In GetViewNetwork API wrapper")

	resp, err := getViewNetworkByID(viewID, id, m)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("VM API returned status %d when reading view network %s for view %s", resp.StatusCode, id, viewID)
	}

	return unpackViewNetworkResponse(resp)
}

// UpdateViewNetwork wraps the PUT call to update a view network.
func UpdateViewNetwork(network *structs.ViewNetworkInfo, m map[string]string) error {
	log.Printf("! In UpdateViewNetwork API wrapper")

	auth, err := util.GetAuth(m)
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"providerType":       network.ProviderType,
		"providerInstanceId": network.ProviderInstanceId,
		"networkId":          network.NetworkId,
		"name":               network.Name,
		"teamIds":            network.TeamIds,
	}

	asJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := util.GetVmApiUrl(m) + "views/" + network.ViewID + "/networks/" + network.ID
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(asJSON))
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Bearer "+auth)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("VM API returned status %d when updating view network %s for view %s", resp.StatusCode, network.ID, network.ViewID)
	}

	log.Printf("! ViewNetwork %s updated", network.ID)
	return nil
}

// DeleteViewNetwork wraps the DELETE call to remove a view network.
func DeleteViewNetwork(viewID, id string, m map[string]string) error {
	log.Printf("! In DeleteViewNetwork API wrapper")

	auth, err := util.GetAuth(m)
	if err != nil {
		return err
	}

	url := util.GetVmApiUrl(m) + "views/" + viewID + "/networks/" + id
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Bearer "+auth)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("VM API returned status %d when deleting view network %s for view %s", resp.StatusCode, id, viewID)
	}

	log.Printf("! ViewNetwork %s deleted", id)
	return nil
}

// ViewNetworkExists returns true if a view network with the given ID exists.
func ViewNetworkExists(viewID, id string, m map[string]string) (bool, error) {
	log.Printf("! In ViewNetworkExists")

	resp, err := getViewNetworkByID(viewID, id, m)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode != http.StatusNotFound, nil
}

// -------------------- Helper functions --------------------

func getViewNetworkByID(viewID, id string, m map[string]string) (*http.Response, error) {
	auth, err := util.GetAuth(m)
	if err != nil {
		return nil, err
	}

	url := util.GetVmApiUrl(m) + "views/" + viewID + "/networks/" + id
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+auth)

	client := &http.Client{}
	return client.Do(req)
}

func unpackViewNetworkResponse(resp *http.Response) (*structs.ViewNetworkInfo, error) {
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)

	asMap := make(map[string]interface{})
	err := json.Unmarshal(buf.Bytes(), &asMap)
	if err != nil {
		return nil, err
	}

	log.Printf("! ViewNetwork response data: %v", asMap)

	var teamIds []string
	if asMap["teamIds"] != nil {
		teams := asMap["teamIds"].([]interface{})
		converted := util.ToStringSlice(&teams)
		teamIds = *converted
	} else {
		teamIds = []string{}
	}

	return &structs.ViewNetworkInfo{
		ID:                 asMap["id"].(string),
		ViewID:             asMap["viewId"].(string),
		ProviderType:       fmt.Sprintf("%v", asMap["providerType"]),
		ProviderInstanceId: asMap["providerInstanceId"].(string),
		NetworkId:          asMap["networkId"].(string),
		Name:               asMap["name"].(string),
		TeamIds:            teamIds,
	}, nil
}
