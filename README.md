PART I (Given for MAC - Check the reference link provided for Linux)
-------------------
Prerequisite

Reference Link
https://hyperledger-fabric.readthedocs.io/en/release-2.5/prereqs.html

# Brew
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install.sh)"
brew --version

# Git
brew install git
git --version

# cURL
brew install curl
curl --version

# Docker and Docker Compose
brew install --cask --appdir="/Applications" docker
open /Applications/Docker.app
docker --version
docker-compose --version

# Go
brew install go@1.21

export PATH="/usr/local/opt/go@1.21/bin:$PATH"
echo 'export PATH="/usr/local/opt/go@1.21/bin:$PATH"' >> ~/.zshrc
go version

# Jq
brew install jq
jq --version

-----------------------
Part II
Reference:
https://hyperledger-fabric.readthedocs.io/en/latest/private_data_tutorial.html#deploy-the-private-data-smart-contract-to-the-channel

NOTE: Make sure your Docker app is running in the background

# To close existing network

cd ./fabric-samples/test-network
./network.sh down
docker ps -a

# To Start the network

./network.sh up createChannel -ca -s couchdb

# To Deploy the Private Data Smart Contract to the channel

./network.sh deployCC -ccn private -ccp ../cro/ -ccl go -ccep "OR('Org1MSP.peer','Org2MSP.peer')" -cccg ../cro/collections_config.json

--------------------------
# To Register Identities
## To Set Environment variables to the Fabric CA client Home

export PATH=${PWD}/../bin:${PWD}:$PATH
export FABRIC_CFG_PATH=$PWD/../config/
export FABRIC_CA_CLIENT_HOME=${PWD}/organizations/peerOrganizations/org1.example.com/
fabric-ca-client register --caname ca-org1 --id.name owner --id.secret ownerpw --id.type client --tls.certfiles "${PWD}/organizations/fabric-ca/org1/tls-cert.pem"
fabric-ca-client enroll -u https://owner:ownerpw@localhost:7054 --caname ca-org1 -M "${PWD}/organizations/peerOrganizations/org1.example.com/users/owner@org1.example.com/msp" --tls.certfiles "${PWD}/organizations/fabric-ca/org1/tls-cert.pem"
cp "${PWD}/organizations/peerOrganizations/org1.example.com/msp/config.yaml" "${PWD}/organizations/peerOrganizations/org1.example.com/users/owner@org1.example.com/msp/config.yaml"
export FABRIC_CA_CLIENT_HOME=${PWD}/organizations/peerOrganizations/org2.example.com/
fabric-ca-client register --caname ca-org2 --id.name buyer --id.secret buyerpw --id.type client --tls.certfiles "${PWD}/organizations/fabric-ca/org2/tls-cert.pem"
fabric-ca-client enroll -u https://buyer:buyerpw@localhost:8054 --caname ca-org2 -M "${PWD}/organizations/peerOrganizations/org2.example.com/users/buyer@org2.example.com/msp" --tls.certfiles "${PWD}/organizations/fabric-ca/org2/tls-cert.pem"
cp "${PWD}/organizations/peerOrganizations/org2.example.com/msp/config.yaml" "${PWD}/organizations/peerOrganizations/org2.example.com/users/buyer@org2.example.com/msp/config.yaml"

--------------------------
# Configuring Peer before executing the smart contract functions 

export PATH=${PWD}/../bin:$PATH
export FABRIC_CFG_PATH=$PWD/../config/
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_LOCALMSPID=Org1MSP
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org1.example.com/users/owner@org1.example.com/msp
export CORE_PEER_ADDRESS=localhost:7051


------------------------
# Sample Inputs

## AddRecord

export RECORD_PROPERTIES=$(echo -n "{\"ObjectType\":\"producer\",\"RecordID\":\"101\",\"ISONumbers\":[\"1234567\",\"789012\"],\"CreatedAtUTC\":112233,\"PremiseID\":\"GHI789\",\"DocumentType\":\"tag_activation\",\"Fields\":{\"date_of_tagging\": \"2024-05-08\"}}" | base64 | tr -d \\n)
peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" -C mychannel -n private -c '{"function":"AddRecord","Args":[]}' --transient "{\"record_properties\":\"$RECORD_PROPERTIES\"}"

export RECORD_PROPERTIES=$(echo -n "{\"ObjectType\":\"distributor\",\"RecordID\":\"102\",\"ISONumbers\":[\"789012\",\"987654\"],\"CreatedAtUTC\":112234,\"PremiseID\":\"ABC123\",\"DocumentType\":\"move_in\",\"Fields\":{\"departure_pid\": \"XYZ789\",\"destination_pid\": \"LMN012\",\"departure_date\": \"2024-05-06\",\"arrival_date\": \"2024-05-07\",\"transportation_license_number\": \"T123456\"}}" | base64 | tr -d \\n)
peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" -C mychannel -n private -c '{"function":"AddRecord","Args":[]}' --transient "{\"record_properties\":\"$RECORD_PROPERTIES\"}"

export RECORD_PROPERTIES=$(echo -n "{\"ObjectType\":\"distributor\",\"RecordID\":\"103\",\"ISONumbers\":[\"321098\",\"456789\"],\"CreatedAtUTC\":112235,\"PremiseID\":\"DEF456\",\"DocumentType\":\"tag_replacement\",\"Fields\":{\"previous_iso_number\": \"987654\",\"transportation_license_number\": \"T789012\"}}" | base64 | tr -d \\n)
peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" -C mychannel -n private -c '{"function":"AddRecord","Args":[]}' --transient "{\"record_properties\":\"$RECORD_PROPERTIES\"}"

export RECORD_PROPERTIES=$(echo -n "{\"ObjectType\":\"producer\",\"RecordID\":\"104\",\"ISONumbers\":[\"321098\",\"890123\"],\"CreatedAtUTC\":112236,\"PremiseID\":\"JKL345\",\"DocumentType\":\"move_out\",\"Fields\":{\"departure_pid\": \"PQR901\",\"destination_pid\": \"STU234\",\"departure_date\": \"2024-05-03\",\"arrival_date\": \"2024-05-04\",\"transportation_license_number\": \"T234567\"}}" | base64 | tr -d \\n)
peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" -C mychannel -n private -c '{"function":"AddRecord","Args":[]}' --transient "{\"record_properties\":\"$RECORD_PROPERTIES\"}"



## GetRecord

peer chaincode query -C mychannel -n private -c '{"function":"GetRecord","Args":["101"]}'



## RevokeRecord

peer chaincode invoke -o localhost:7050 \
--ordererTLSHostnameOverride orderer.example.com \
--tls --cafile "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" \
-C mychannel -n private -c '{ "function": "RevokeRecord", "Args": ["101", "Incorrect information"] }'



## GetRecords

peer chaincode query -C mychannel -n private -c '{"function":"GetRecords","Args":["789012", "112233", "112237", "10"]}'



---------------------------
# To Switch between Peers



export CORE_PEER_LOCALMSPID=Org1MSP
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org1.example.com/users/owner@org1.example.com/msp
export CORE_PEER_ADDRESS=localhost:7051





export CORE_PEER_LOCALMSPID=Org2MSP
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org2.example.com/users/buyer@org2.example.com/msp
export CORE_PEER_ADDRESS=localhost:9051