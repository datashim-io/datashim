package dataset

import (
	"context"
	"strings"

	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"k8s.io/apimachinery/pkg/api/errors"
	
	"github.com/akolb1/gometastore/hmsclient"
)

var log = logf.Log.WithName("metastore_client")

func processCatalogEntry(catalogUri string, table string) (CatalogEntry c,error) {
	
	catalogLogger := log.WithValues("catalogUri",catalogUri)
	catalogLogger.Info("Querying Catalog "+catalogUri)

	catHostPort = string.Split(catalogUri,":")

	//if no port is given, assume standard Hive Metastore Port of 9083
	if len(catHostPort) == 1 {
		catHost = catHostPort[0]
		catPort = 9083
	} else if len(catHostPort) == 2 {
		catHost = catHostPort[0]
		catPort = catHostPort[1]
	} else {
		catalogLogger.Error("CatalogURI cannot be parsed.. quitting")
		return nil, errors.New("CatalogURI is in incorrect format")
	}
	
	catalogLogger.Info("Catalog Host "+catHost+" Catalog Port "+catPort)
		
	hiveclient, err := hmsclient.Open(catHost, catPort)
	if err != nil {
		catalogLogger.Error(err)
		return CatalogEntry{}, error
	}
	defer hiveclient.Close()
	
	//We are assuming that the table entry will be in the form <db-name>/<table-name>
	catDBTable = string.split(table, "/")
	//If there is no / in the table input, we'll assume that the database name is 'default'
	if len(catDBTable) == 1 {
		dbName = "default"
		tableName = catDBTable[0]
	} else if len(catDBTable) == 2 {
		dbName = catDBTable[0]
		tableName = catDBTable[1]
	} else {
		catalogLogger.Error("Table name cannot be parsed..")
		return nil, errors.New("Table name is in incorrect format ")
	}

	catalogLogger.Info("looking up "+dbName+"/"+tableName)

	hiveTbl, err := hiveclient.GetTableName(dbName, tableName)
	
	if err != nil || hiveTbl == nil {
		catalogLogger.Error("Table could not be found in the catalog "+err)
		return nil, errors.New("Table could not be found in the catalog") 
	}

	catalogLogger.Info("finding partitions for "+dbName+"/"+tableName)

	tblPartNames, err := hiveclient.GetPartitionNames(dbName, tableName)
	
	if err != nil {
		//Something could go wrong, we don't know what it is now
		catalogLogger.Error("Could not lookup partition names for "+dbName+"/"+tableName)
		//Let's not return a nil for now
		//return nil, errors.New("Table partitions could not be retrieved")
	}
	
	locations := []string
	if len(tblPartNames)==0 {
		catalogLogger.Info("Table "+dbName+"/"+tableName+" is not partitioned")
		//Get the location from the table properties, if present.

		table, _ := hiveclient.GetTable(dbName, tableName)
		loc := table.GetSd().GetLocation()
		catalogLogger.Info("Location for this table is "+loc)
		if strings.HasPrefix(loc, "s3a:") {
			locations.append(loc)
		}
	} else {
		tablePartitions, err =  hiveclient.GetPartitionsByName(dbName, tableName, tblpartNames)
	    if err != nil {
			//Something reaaaally went wromg
			catalogLogger.Error("Could not look up Partition info for "+dbName+"/"+tableName)
			return nil, errors.New("Could not look up Partition info")
		}
		for _, part := range partitions {
			loc := part.GetSd().GetLocation()
			if strings.HasPrefix(loc, "s3a:") {
				locations.append(loc)
			}
		}

	}

	return locations, nil
}
