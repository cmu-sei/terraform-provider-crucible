/*
Crucible
Copyright 2020 Carnegie Mellon University.
NO WARRANTY. THIS CARNEGIE MELLON UNIVERSITY AND SOFTWARE ENGINEERING INSTITUTE MATERIAL IS FURNISHED ON AN "AS-IS" BASIS. CARNEGIE MELLON UNIVERSITY MAKES NO WARRANTIES OF ANY KIND, EITHER EXPRESSED OR IMPLIED, AS TO ANY MATTER INCLUDING, BUT NOT LIMITED TO, WARRANTY OF FITNESS FOR PURPOSE OR MERCHANTABILITY, EXCLUSIVITY, OR RESULTS OBTAINED FROM USE OF THE MATERIAL. CARNEGIE MELLON UNIVERSITY DOES NOT MAKE ANY WARRANTY OF ANY KIND WITH RESPECT TO FREEDOM FROM PATENT, TRADEMARK, OR COPYRIGHT INFRINGEMENT.
Released under a MIT (SEI)-style license, please see license.txt or contact permission@sei.cmu.edu for full terms.
[DISTRIBUTION STATEMENT A] This material has been approved for public release and unlimited distribution.  Please see Copyright notice for non-US Government use and distribution.
Carnegie Mellon(R) and CERT(R) are registered in the U.S. Patent and Trademark Office by Carnegie Mellon University.
DM20-0181
*/

package util

import (
	"context"

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

