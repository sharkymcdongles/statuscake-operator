/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"

	srev1alpha1 "https://github.com/sharkymcdongles/statuscake-operator/api/v1alpha1"
	"github.com/StatusCakeDev/statuscake-go"
	"github.com/StatusCakeDev/statuscake-go/credentials"
)

// We create a struct to store the UptimeTest configuration paramaters from the API provided the test already exists
type UptimeTestAPISpec struct {
	Name            string
	TestType        statuscake.UptimeTestType
	WebsiteUrl      string
	CheckRate       statuscake.UptimeTestCheckRate
	Confirmation    int32
	ContactGroups   []string
	CustomHeader    map[string]string
	EnableSslAlert  bool
	FollowRedirects bool
	ForceHttps      bool
	Paused          bool
	Tags            []string
	Timeout         int32
	TriggerRate     int32
	UserAgent       string
}

// Create new UptimeTest struct from the values within the CRD to use for comparison later to ensure type safety
// TODO: Dedeuplicate Tags and ContactGroups due to CRD validation for uniqueitems not being supported for slices
func NewUptimeTestAPISpecFromCRD(testSpec srev1alpha1.UptimeTestSpec, checkName string) *UptimeTestAPISpec {
	var testType statuscake.UptimeTestType
	var checkRate statuscake.UptimeTestCheckRate

	testType = statuscake.UptimeTestType(testSpec.TestType)
	checkRate = statuscake.UptimeTestCheckRate(testSpec.CheckRate)

	// Fallback to HTTP TestType if TestType is invalid
	// TODO: Consider logging this or expand on how to handle. TestType is enforced via enum on the CRD, but maybe there is a way this could mess up.
	if !testType.Valid() {
		testType = statuscake.UptimeTestTypeHTTP
	}

	// Fallback to 5 minute CheckRate if CheckRate is invalid
	// TODO: Consider logging this or expand on how to handle. TestType is enforced via enum on the CRD, but maybe there is a way this could mess up.
	if !checkRate.Valid() {
		checkRate = statuscake.UptimeTestCheckRateFiveMinutes
	}

	u := UptimeTestAPISpec{
		Name:            checkName,
		TestType:        testType,
		WebsiteUrl:      testSpec.WebsiteUrl,
		CheckRate:       checkRate,
		Confirmation:    int32(testSpec.Confirmation),
		ContactGroups:   testSpec.ContactGroups,
		CustomHeader:    testSpec.CustomHeader,
		EnableSslAlert:  testSpec.EnableSslAlert,
		FollowRedirects: testSpec.FollowRedirects,
		ForceHttps:      testSpec.ForceHttps,
		Paused:          testSpec.Paused,
		Tags:            testSpec.Tags,
		Timeout:         int32(testSpec.Timeout),
		TriggerRate:     int32(testSpec.TriggerRate),
		UserAgent:       testSpec.UserAgent,
	}

	return &u
}

// Create StatusCake API client in memory
func StatusCakeClient(apiToken string) *statuscake.Client {
	bearer := credentials.NewBearerWithStaticToken(apiToken)
	client := statuscake.NewClient(statuscake.WithRequestCredentials(bearer))
	return client
}

// Create UptimeTest
func CreateUptimeTest(client *statuscake.Client, uptimeTest *UptimeTestAPISpec) (statuscake.APIResponse, error) {

	res, err := client.CreateUptimeTest(context.Background()).
		Name(uptimeTest.Name).
		TestType(statuscake.UptimeTestType(uptimeTest.TestType)).
		WebsiteURL(uptimeTest.WebsiteUrl).
		CheckRate(uptimeTest.CheckRate).
		Confirmation(int32(uptimeTest.Confirmation)).
		ContactGroups(uptimeTest.ContactGroups).
		CustomHeader(fmt.Sprint(uptimeTest.CustomHeader)).
		EnableSSLAlert(uptimeTest.EnableSslAlert).
		FollowRedirects(uptimeTest.FollowRedirects).
		Paused(uptimeTest.Paused).
		Tags(uptimeTest.Tags).
		Timeout(int32(uptimeTest.Timeout)).
		TriggerRate(int32(uptimeTest.TriggerRate)).
		UserAgent(uptimeTest.UserAgent).
		Execute()

	return res, err
}

func DeleteUptimeTest(client *statuscake.Client, testId string) error {
	err := client.DeleteUptimeTest(context.Background(), testId).Execute()
	return err
}

// Get StatusCake UptimeTest
func GetUptimeTest(client *statuscake.Client, testId string) (*UptimeTestAPISpec, error) {
	uptimeTest, err := client.GetUptimeTest(context.Background(), testId).Execute()

	userAgent := "Mozilla/5.0 (Windows NT 6.2; WOW64) AppleWebKit/537.4 (KHTML, like Gecko) Chrome/98 Safari/537.4 (StatusCake)"

	if uptimeTest.Data.UserAgent != nil {
		userAgent = *uptimeTest.Data.UserAgent
	}

	u := UptimeTestAPISpec{
		Name:            uptimeTest.Data.Name,
		TestType:        uptimeTest.Data.TestType,
		WebsiteUrl:      uptimeTest.Data.WebsiteURL,
		CheckRate:       uptimeTest.Data.CheckRate,
		Confirmation:    uptimeTest.Data.Confirmation,
		ContactGroups:   uptimeTest.Data.ContactGroups,
		EnableSslAlert:  uptimeTest.Data.EnableSSLAlert,
		FollowRedirects: uptimeTest.Data.FollowRedirects,
		Paused:          uptimeTest.Data.Paused,
		Tags:            uptimeTest.Data.Tags,
		Timeout:         uptimeTest.Data.Timeout,
		TriggerRate:     uptimeTest.Data.TriggerRate,
		UserAgent:       userAgent,
	}

	return &u, err
}

// Update StatusCake UptimeTest
func UpdateUptimeTest(client *statuscake.Client, testId string, expectedState *UptimeTestAPISpec) error {

	err := client.UpdateUptimeTest(context.Background(), testId).
		CheckRate(expectedState.CheckRate).
		Confirmation(expectedState.Confirmation).
		ContactGroups(expectedState.ContactGroups).
		EnableSSLAlert(expectedState.EnableSslAlert).
		FollowRedirects(expectedState.FollowRedirects).
		Paused(expectedState.Paused).
		Tags(expectedState.Tags).
		Timeout(expectedState.Timeout).Timeout(expectedState.Timeout).
		TriggerRate(expectedState.TriggerRate).
		UserAgent(expectedState.UserAgent).
		Execute()

	return err
}
