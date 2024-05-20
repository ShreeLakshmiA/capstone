/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

const recordCollection = "recordCollection"

// CROContract of this fabric sample
type CROContract struct {
	contractapi.Contract
}

// CSFRecord describes main record details that are visible to all organizations
type CSFRecord struct {
	ObjectType       string   `json:"objectType"`       // Helps in categorizing objects in the state database
	RecordID         string   `json:"recordId"`         // Unique Identifier for the record
	ISONumbers       []string `json:"isoNumbers"`       // List of unique identifiers corresponding to each animal
	CreatedAtUTC     uint     `json:"createdAtUTC"`     // Timestamp when the record was created
	PremiseID        string   `json:"premiseId"`        // The premise associated with the animals
	DocumentType     string   `json:"documentType"`     // Type of document (e.g., tag_activation, tag_replacement)
	Revoked          bool     `json:"revoked"`          // Indicates whether the record is revoked
	RevocationReason string   `json:"revocationReason"` // Reason for revocation, if applicable
}

// CSFRecordPrivateDetails describes details that are private to owners
type CSFRecordPrivateDetails struct {
	RecordID string            `json:"recordId"` // Links back to the public record
	Fields   map[string]string `json:"fields"`   // Sensitive information based on the document type
}

type Record struct {
	ObjectType       string            `json:"objectType"`       // Helps in categorizing objects in the state database
	RecordID         string            `json:"recordId"`         // Unique Identifier for the record
	ISONumbers       []string          `json:"isoNumbers"`       // List of unique identifiers corresponding to each animal
	CreatedAtUTC     uint              `json:"createdAtUTC"`     // Timestamp when the record was created
	PremiseID        string            `json:"premiseId"`        // The premise associated with the animals
	DocumentType     string            `json:"documentType"`     // Type of document (e.g., tag_activation, tag_replacement)
	Fields           map[string]string `json:"fields"`           // Sensitive information based on the document type
	Revoked          bool              `json:"revoked"`          // Indicates whether the record is revoked
	RevocationReason string            `json:"revocationReason"` // Reason for revocation, if applicable
}

// CreateCSFRecord creates a new record by placing the main record details in the recordCollection
// that can be read by both organizations. The private details are stored in the owner's org specific collection.
func (s *CROContract) AddRecord(ctx contractapi.TransactionContextInterface) error {

	// Get new record from transient map
	transientMap, err := ctx.GetStub().GetTransient()
	if err != nil {
		return fmt.Errorf("error getting transient: %v", err)
	}

	// Record properties are private, therefore they get passed in transient field, instead of func args
	transientRecordJSON, ok := transientMap["record_properties"]
	if !ok {
		// log error to stdout
		return fmt.Errorf("record not found in the transient map input")
	}

	type recordTransientInput struct {
		ObjectType       string            `json:"objectType"`       // Helps in categorizing objects in the state database
		RecordID         string            `json:"recordId"`         // Unique Identifier for the record
		ISONumbers       []string          `json:"isoNumbers"`       // List of unique identifiers corresponding to each animal
		CreatedAtUTC     uint              `json:"createdAtUTC"`     // Timestamp when the record was created
		PremiseID        string            `json:"premiseId"`        // The premise associated with the animals
		DocumentType     string            `json:"documentType"`     // Type of document (e.g., tag_activation, tag_replacement)
		Revoked          bool              `json:"revoked"`          // Indicates whether the record is revoked
		RevocationReason string            `json:"revocationReason"` // Reason for revocation, if applicable
		Fields           map[string]string `json:"fields"`           // Sensitive information based on the document type
	}

	var recordInput recordTransientInput
	err = json.Unmarshal(transientRecordJSON, &recordInput)
	if err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	if len(recordInput.ObjectType) == 0 {
		return fmt.Errorf("objectType field must be a non-empty string")
	}
	if len(recordInput.RecordID) == 0 {
		return fmt.Errorf("recordId field must be a non-empty string")
	}
	if len(recordInput.ISONumbers) == 0 {
		return fmt.Errorf("isoNumbers field must be a non-empty array")
	}
	if recordInput.CreatedAtUTC == 0 {
		return fmt.Errorf("createdAtUTC field must be a non-empty uint")
	}
	if len(recordInput.PremiseID) == 0 {
		return fmt.Errorf("premiseId field must be a non-empty string")
	}
	if len(recordInput.DocumentType) == 0 {
		return fmt.Errorf("documentType field must be a non-empty string")
	}

	// Check if record already exists
	recordAsBytes, err := ctx.GetStub().GetPrivateData(recordCollection, recordInput.RecordID)
	if err != nil {
		return fmt.Errorf("failed to get record: %v", err)
	} else if recordAsBytes != nil {
		fmt.Println("Record already exists: " + recordInput.RecordID)
		return fmt.Errorf("this record already exists: " + recordInput.RecordID)
	}

	// Get ID of submitting client identity
	clientID, err := submittingClientIdentity(ctx)
	if err != nil {
		return err
	}

	// Verify that the client is submitting request to peer in their organization
	err = verifyClientOrgMatchesPeerOrg(ctx)
	if err != nil {
		return fmt.Errorf("CreateCSFRecord cannot be performed: Error %v", err)
	}

	// Make submitting client the owner
	record := CSFRecord{
		ObjectType:       recordInput.ObjectType,
		RecordID:         recordInput.RecordID,
		ISONumbers:       recordInput.ISONumbers,
		CreatedAtUTC:     recordInput.CreatedAtUTC,
		PremiseID:        recordInput.PremiseID,
		DocumentType:     recordInput.DocumentType,
		Revoked:          false,
		RevocationReason: "",
	}
	recordJSONasBytes, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal record into JSON: %v", err)
	}

	// Save record to private data collection
	log.Printf("CreateCSFRecord Put: collection %v, ID %v, owner %v", recordCollection, recordInput.RecordID, clientID)

	err = ctx.GetStub().PutPrivateData(recordCollection, recordInput.RecordID, recordJSONasBytes)
	if err != nil {
		return fmt.Errorf("failed to put record into private data collection: %v", err)
	}

	// Save record details to collection visible to owning organization
	recordPrivateDetails := CSFRecordPrivateDetails{
		RecordID: recordInput.RecordID,
		Fields:   recordInput.Fields,
	}

	recordPrivateDetailsAsBytes, err := json.Marshal(recordPrivateDetails)
	if err != nil {
		return fmt.Errorf("failed to marshal into JSON: %v", err)
	}

	// Get collection name for this organization.
	orgCollection, err := getCollectionName(ctx)
	if err != nil {
		return fmt.Errorf("failed to infer private collection name for the org: %v", err)
	}

	// Put record private details into owner's org specific private data collection
	log.Printf("Put: collection %v, ID %v", orgCollection, recordInput.RecordID)
	err = ctx.GetStub().PutPrivateData(orgCollection, recordInput.RecordID, recordPrivateDetailsAsBytes)
	if err != nil {
		return fmt.Errorf("failed to put record private details: %v", err)
	}
	return nil
}

// getRecord retrieves a record by its RecordID
func (s *CROContract) GetRecord(ctx contractapi.TransactionContextInterface, recordID string) (*Record, error) {
	if len(recordID) == 0 {
		return nil, fmt.Errorf("recordId field must be a non-empty string")
	}

	// Verify that the client is submitting request to peer in their organization
	// err := verifyClientOrgMatchesPeerOrg(ctx)
	// if err != nil {
	// 	return nil, fmt.Errorf("getRecord cannot be performed: Error %v", err)
	// }

	log.Printf("GetRecord: collection %v, ID %v", recordCollection, recordID)
	recordJSON, err := ctx.GetStub().GetPrivateData(recordCollection, recordID) //get the record from chaincode state
	if err != nil {
		return nil, fmt.Errorf("failed to read record: %v", err)
	}

	// No Record found, return empty response
	if recordJSON == nil {
		log.Printf("%v does not exist in collection %v", recordID, recordCollection)
		return nil, nil
	}

	var csfRecord *CSFRecord
	err = json.Unmarshal(recordJSON, &csfRecord)

	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	// Make submitting client the owner
	record := Record{
		ObjectType:       csfRecord.ObjectType,
		RecordID:         csfRecord.RecordID,
		ISONumbers:       csfRecord.ISONumbers,
		CreatedAtUTC:     csfRecord.CreatedAtUTC,
		PremiseID:        csfRecord.PremiseID,
		DocumentType:     csfRecord.DocumentType,
		Revoked:          csfRecord.Revoked,
		RevocationReason: csfRecord.RevocationReason,
	}

	// Get the collection name for the caller's organization
	orgCollection, err := getCollectionName(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to infer private collection name for the org: %v", err)
	}

	// Try to get the record from the private collection
	recordAsBytes, err := ctx.GetStub().GetPrivateData(orgCollection, recordID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve record: %v", err)
	}
	if recordAsBytes == nil {
		record.Fields = make(map[string]string)
		return &record, nil
	}

	var recordPrivate CSFRecordPrivateDetails
	err = json.Unmarshal(recordAsBytes, &recordPrivate)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal record: %v", err)
	}
	record.Fields = recordPrivate.Fields

	return &record, nil

}

// GetRecords retrieves records with matching ISONumbers from the caller's organization collection
// with optional date filtering and record limit.
func (s *CROContract) GetRecords(ctx contractapi.TransactionContextInterface, ISONumbers string, DateFrom uint, DateTo uint, Limit uint) ([]*Record, error) {
	if len(ISONumbers) == 0 {
		return nil, fmt.Errorf("isoNumbers field must be a non-empty string")
	}

	// Verify that the client is submitting request to peer in their organization
	// err := verifyClientOrgMatchesPeerOrg(ctx)
	// if err != nil {
	// 	return nil, fmt.Errorf("getRecords cannot be performed: Error %v", err)
	// }

	// Construct query for records with the matching ISONumbers
	// Initialize query selector
	selector := map[string]interface{}{
		"selector": map[string]interface{}{
			"isoNumbers": map[string]interface{}{"$in": []string{ISONumbers}},
		},
	}

	// Add date filters if provided
	if DateFrom != 0 && DateTo != 0 {
		selector["selector"].(map[string]interface{})["createdAtUTC"] = map[string]interface{}{
			"$gte": DateFrom,
			"$lte": DateTo,
		}
	}

	// Marshal selector into JSON format
	queryBytes, err := json.Marshal(selector)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal selector into JSON format: %v", err)
	}

	// Execute the query
	resultsIterator, err := ctx.GetStub().GetPrivateDataQueryResult(recordCollection, string(queryBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %v", err)
	}
	defer resultsIterator.Close()

	var records []*Record
	counter := uint(0)

	// Parse query results into RecordInfo array
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, fmt.Errorf("error while iterating over query results: %v", err)
		}

		var csfRecord CSFRecord
		err = json.Unmarshal(queryResponse.Value, &csfRecord)
		if err != nil {
			return nil, fmt.Errorf("error while unmarshalling record JSON: %v", err)
		}

		// Make submitting client the owner
		record := Record{
			ObjectType:       csfRecord.ObjectType,
			RecordID:         csfRecord.RecordID,
			ISONumbers:       csfRecord.ISONumbers,
			CreatedAtUTC:     csfRecord.CreatedAtUTC,
			PremiseID:        csfRecord.PremiseID,
			DocumentType:     csfRecord.DocumentType,
			Revoked:          csfRecord.Revoked,
			RevocationReason: csfRecord.RevocationReason,
		}

		// Get the collection name for the caller's organization
		orgCollection, err := getCollectionName(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to infer private collection name for the org: %v", err)
		}

		// Try to get the record from the private collection
		recordAsBytes, err := ctx.GetStub().GetPrivateData(orgCollection, csfRecord.RecordID)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve record: %v", err)
		}
		if recordAsBytes == nil {
			record.Fields = make(map[string]string)
			// Append record information to results array
			records = append(records, &record)
		} else {
			var recordPrivate CSFRecordPrivateDetails
			err = json.Unmarshal(recordAsBytes, &recordPrivate)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal record: %v", err)
			}
			record.Fields = recordPrivate.Fields

			// Append record information to results array
			records = append(records, &record)
		}

		// Increment counter and check limit
		counter++
		if Limit > 0 && counter >= Limit {
			break
		}
	}

	return records, nil
}

// revokeRecord sets revoked to true and sets the RevocationReason of the record corresponding to the RecordID
func (s *CROContract) RevokeRecord(ctx contractapi.TransactionContextInterface, RecordID string, RevocationReason string) error {
	if len(RecordID) == 0 {
		return fmt.Errorf("recordId field must be a non-empty string")
	}

	// Verify that the client is submitting request to peer in their organization
	// err := verifyClientOrgMatchesPeerOrg(ctx)
	// if err != nil {
	// 	return fmt.Errorf("revokeRecord cannot be performed: Error %v", err)
	// }

	// Get the record from the private collection
	recordAsBytes, err := ctx.GetStub().GetPrivateData(recordCollection, RecordID)
	if err != nil {
		return fmt.Errorf("failed to get record from record collection: %v", err)
	}
	if recordAsBytes == nil {
		return fmt.Errorf("record does not exist: %s", RecordID)
	}

	// Unmarshal the record
	var record CSFRecord
	err = json.Unmarshal(recordAsBytes, &record)
	if err != nil {
		return fmt.Errorf("failed to unmarshal record: %v", err)
	}

	if !record.Revoked {
		// Update the record
		record.Revoked = true
		record.RevocationReason = RevocationReason

		// Marshal the updated record
		recordJSONasBytes, err := json.Marshal(record)
		if err != nil {
			return fmt.Errorf("failed to marshal record into JSON: %v", err)
		}

		// Put the updated record back into the private data collection
		err = ctx.GetStub().PutPrivateData(recordCollection, RecordID, recordJSONasBytes)
		if err != nil {
			return fmt.Errorf("failed to put updated record into private data collection: %v", err)
		}

		return nil
	} else {
		return fmt.Errorf("record already revoked")
	}

}

// getCollectionName is an internal helper function to get collection of submitting client identity.
func getCollectionName(ctx contractapi.TransactionContextInterface) (string, error) {

	// Get the MSP ID of submitting client identity
	clientMSPID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return "", fmt.Errorf("failed to get verified MSPID: %v", err)
	}

	// Create the collection name
	orgCollection := clientMSPID + "PrivateCollection"

	return orgCollection, nil
}

// verifyClientOrgMatchesPeerOrg is an internal function used verify client org id and matches peer org id.
func verifyClientOrgMatchesPeerOrg(ctx contractapi.TransactionContextInterface) error {
	clientMSPID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("failed getting the client's MSPID: %v", err)
	}
	peerMSPID, err := shim.GetMSPID()
	if err != nil {
		return fmt.Errorf("failed getting the peer's MSPID: %v", err)
	}

	if clientMSPID != peerMSPID {
		return fmt.Errorf("client from org %v is not authorized to read or write private data from an org %v peer", clientMSPID, peerMSPID)
	}

	return nil
}

func submittingClientIdentity(ctx contractapi.TransactionContextInterface) (string, error) {
	b64ID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return "", fmt.Errorf("failed to read clientID: %v", err)
	}
	decodeID, err := base64.StdEncoding.DecodeString(b64ID)
	if err != nil {
		return "", fmt.Errorf("failed to base64 decode clientID: %v", err)
	}
	return string(decodeID), nil
}

func main() {
	cc, err := contractapi.NewChaincode(&CROContract{})
	if err != nil {
		log.Panicf("Error creating CROChaincode chaincode: %v", err)
	}

	if err := cc.Start(); err != nil {
		log.Panicf("Error starting CROChaincode chaincode: %v", err)
	}
}
