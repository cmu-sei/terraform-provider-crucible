// Copyright 2022 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package util

import (
	"context"
	"log"
	"strings"

	"golang.org/x/oauth2"
)

// Helper functions used throughout provider

// ToStringSlice converts a slice of empty interfaces to a slice of strings. Go won't let us do this implicitly.
func ToStringSlice(data *[]interface{}) *[]string {
	var converted []string
	for _, entry := range *data {
		converted = append(converted, entry.(string))
	}
	return &converted
}

// GetAuth gets an auth token.
func GetAuth(m map[string]string) (string, error) {
	con := &oauth2.Config{
		ClientID:     m["client_id"],
		ClientSecret: m["client_secret"],
		Endpoint: oauth2.Endpoint{
			AuthURL:  m["auth_url"],
			TokenURL: m["player_token_url"],
		},
	}

	tok, err := con.PasswordCredentialsToken(context.Background(), m["username"], m["password"])
	if err != nil {
		return "", err
	}
	return tok.AccessToken, nil
}

// PairInList returns true if a given key/value pair exists somewhere in a list of maps
func PairInList(list []interface{}, key, value string) bool {
	for _, curr := range list {
		asMap := curr.(map[string]interface{})
		if asMap[key] == value {
			return true
		}
	}
	return false
}

// Ternary returns a if the condition is true and b otherwise. Go doesn't have an actual ternary operator.
//
// param condition: The condition to evaluate
//
// param a: The value to return if condition is true
//
// param b: The value to return if condition is false
func Ternary(condition bool, a, b interface{}) interface{} {
	if condition {
		return a
	}
	return b
}

// StrSliceContains returns true if a string slice contains a given string.
//
// param arr: The slice to look in
//
// param str: The string to look for
//
// Returns true if str is in arr and false if not
func StrSliceContains(arr *[]string, str string) bool {
	for _, elem := range *arr {
		if elem == str {
			return true
		}
	}
	return false
}

// Returns the normalized url for the player api
func GetPlayerApiUrl(m map[string]string) string {
	return GetApiUrl(m, "player_api_url")
}

// Returns the normalized url for the vm api
func GetVmApiUrl(m map[string]string) string {
	return GetApiUrl(m, "vm_api_url")
}

// Returns the normalized url for the caster api
func GetCasterApiUrl(m map[string]string) string {
	return GetApiUrl(m, "caster_api_url")
}

// GetApiUrl returns a url from the settings map, normalized to end in /api/
//
// param m: The settings map
//
// param urlName: The name of the url setting in the map
//
// Returns empty string is urlName is not found in the map
func GetApiUrl(m map[string]string, urlName string) string {
	log.Printf("! Getting API Url for %s", urlName)
	if url, exists := m[urlName]; exists {
		log.Printf("! URL = %s", url)
		url = strings.TrimSuffix(url, "/")
		url = strings.TrimSuffix(url, "/api")
		url = url + "/api/"
		log.Printf("! Normalized URL = %s", url)
		return url
	}

	log.Printf("! URL not found")
	return ""
}
