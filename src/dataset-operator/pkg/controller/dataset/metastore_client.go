package dataset

import (
	"errors"
	"strconv"
	"strings"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	"github.com/akolb1/gometastore/hmsclient"
)

var catalogLogger = logf.Log.WithName("metastore_client")

func processCatalogEntry(catalogUri string, table string) ([]string, error) {

	catalogLogger := log.WithValues("catalogUri", catalogUri)
	catalogLogger.Info("Querying Catalog " + catalogUri)

	catHostPort := strings.Split(catalogUri, ":")

	//if no port is given, assume standard Hive Metastore Port of 9083
	var catHost string
	var catPort int
	if len(catHostPort) == 1 {
		catHost = catHostPort[0]
		catPort = 9083
	} else if len(catHostPort) == 2 {
		catHost = catHostPort[0]
		catPort, _ = strconv.Atoi(catHostPort[1])
	} else {
		catalogLogger.Error(nil, "CatalogURI cannot be parsed.. quitting")
		return nil, k8serrors.NewBadRequest("CatalogURI is in incorrect format")
	}

	catalogLogger.Info("Catalog Host : %s, Catalog Port: %d", catHost, catPort)

	hiveclient, err := hmsclient.Open(catHost, catPort)
	if err != nil {
		catalogLogger.Error(err, "could not open connection to metastore")
		return nil, k8serrors.NewInternalError(errors.New("Cannot connect to the metastore : " + catHost))
	}
	defer hiveclient.Close()

	var dbName, tableName string
	//We are assuming that the table entry will be in the form <db-name>/<table-name>
	catDBTable := strings.Split(table, "/")
	//If there is no / in the table input, we'll assume that the database name is 'default'
	if len(catDBTable) == 1 {
		dbName = "default"
		tableName = catDBTable[0]
	} else if len(catDBTable) == 2 {
		dbName = catDBTable[0]
		tableName = catDBTable[1]
	} else {
		catalogLogger.Error(nil, "Table name cannot be parsed..")
		return nil, k8serrors.NewBadRequest("Table name is in incorrect format ")
	}

	catalogLogger.Info("looking up " + dbName + "/" + tableName)

	hiveTbl, err := hiveclient.GetTable(dbName, tableName)

	if err != nil || hiveTbl == nil {
		catalogLogger.Error(err, "Table could not be found in the catalog ")
		return nil, k8serrors.NewBadRequest("Table could not be found in the catalog")
	}

	catalogLogger.Info("finding partitions for " + dbName + "/" + tableName)

	//256 is a magic number but is the number of partitions you want to get from the table
	tblPartNames, err := hiveclient.GetPartitionNames(dbName, tableName, 256)

	if err != nil {
		//Something could go wrong, we don't know what it is now
		catalogLogger.Error(err, "Could not lookup partition names for "+dbName+"/"+tableName)
		//Let's return a nil for now
		return nil, k8serrors.NewInternalError(errors.New("Table partitions could not be retrieved"))
	}

	var locations []string
	if len(tblPartNames) == 0 {
		catalogLogger.Info("Table " + dbName + "/" + tableName + " is not partitioned")
		//Get the location from the table properties, if present.

		table, _ := hiveclient.GetTable(dbName, tableName)
		loc := table.GetSd().GetLocation()
		catalogLogger.Info("Location for this table is " + loc)
		if strings.HasPrefix(loc, "s3a:") {
			locations = append(locations, loc)
		}
	} else {
		tablePartitions, err := hiveclient.GetPartitionsByNames(dbName, tableName, tblPartNames)
		if err != nil {
			//Something reaaaally went wromg
			catalogLogger.Error(err, "Could not look up Partition info for "+dbName+"/"+tableName)
			return nil, k8serrors.NewBadRequest("Could not look up Partition info")
		}
		for _, part := range tablePartitions {
			loc := part.GetSd().GetLocation()
			catalogLogger.Info("Got a location: ", "location", loc)
			if strings.HasPrefix(loc, "s3a:") {
				locations = append(locations, loc)
			}
		}

	}

	return locations, nil
}
