package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/RHEcosystemAppEng/cluster-iq/internal/inventory"
	_ "github.com/RHEcosystemAppEng/cluster-iq/internal/inventory"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// HandlerGetInstances handles the request for obtain the entire Instances list
//	@Summary		Obtain every Instance
//	@Description	Returns a list of Instances with every Instance in the inventory
//	@Tags			Instances
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	InstanceListResponse
//	@Failure		500	{object}	nil
//	@Router			/instances/ [get]
func HandlerGetInstances(c *gin.Context) {
	logger.Debug("Retrieving complete instance inventory")
	addHeaders(c)

	instances, err := getInstances()
	if err != nil {
		logger.Error("Can't retrieve Instances list", zap.Error(err))
		c.PureJSON(http.StatusInternalServerError, nil)
		return
	}

	response := NewInstanceListResponse(instances)
	c.PureJSON(http.StatusOK, response)
}

// HandlerGetInstancesByID handles the request for obtain an Instance by its ID
//	@Summary		Obtain a single Instance by its ID
//	@Description	Returns a list of Instances with a single Instance filtered by ID
//	@Tags			Instances
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	InstanceListResponse
//	@Failure		404	{object}	nil
//	@Failure		500	{object}	nil
//	@Router			/instances/:instance_id [get]
func HandlerGetInstancesByID(c *gin.Context) {
	instanceID := c.Param("instance_id")
	logger.Debug("Retrieving instance by ID", zap.String("instance_id", instanceID))
	addHeaders(c)

	instances, err := getInstanceByID(instanceID)
	if err != nil {
		logger.Error("Instance not found", zap.String("instance_id", instanceID), zap.Error(err))
		c.PureJSON(http.StatusNotFound, nil)
		return
	}

	response := NewInstanceListResponse(instances)
	c.PureJSON(http.StatusOK, response)
}

// HandlerPostInstance handles the request for writting a new Instance in the inventory
//	@Summary		Creates a new Instance in the inventory
//	@Description	Receives and write into the DB the information for a new Instance
//	@Tags			Instances
//	@Accept			json
//	@Produce		json
//	@Param			instance	body		[]inventory.Instance	true	"New Instance to be added"
//	@Success		200			{object}	nil
//	@Failure		500			{object}	nil
//	@Router			/instances [post]
func HandlerPostInstance(c *gin.Context) {
	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		logger.Error("Can't get body from request", zap.Error(err))
		c.PureJSON(http.StatusInternalServerError, nil)
		return
	}

	var instances []inventory.Instance
	err = json.Unmarshal([]byte(body), &instances)
	if err != nil {
		logger.Error("Can't obtain data from body requet", zap.Error(err))
		c.PureJSON(http.StatusBadRequest, nil)
		return
	}

	logger.Debug("Writing a new Instance", zap.Reflect("instance", instances))
	err = writeInstance(instances)
	if err != nil {
		logger.Error("Can't write new instances into DB", zap.Error(err))
		c.PureJSON(http.StatusInternalServerError, nil)
		return
	}
}

// HandlerDeleteInstance handles the request for removing an Instance in the inventory
//	@Summary		Deletes an Instance in the inventory
//	@Description	Deletes an Instance present in the inventory by its ID
//	@Tags			Instances
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	nil
//	@Failure		500	{object}	nil
//	@Router			/instances/:instance_id [delete]
// TODO: Not Implemented
func HandlerDeleteInstance(c *gin.Context) {
	instanceID := c.Param("instance_id")
	logger.Debug("Removing an Instance", zap.String("instance_id", instanceID))
	if err := deleteInstance(instanceID); err != nil {
		logger.Error("Can't delete instance from DB", zap.String("instance_id", instanceID), zap.Error(err))
		c.PureJSON(http.StatusInternalServerError, nil)
		return
	}

	c.PureJSON(http.StatusOK, nil)
}

// HandlerPatchInstance handles the request for patching an Instance in the inventory
//	@Summary		Patches an Instance in the inventory
//	@Description	Receives and patch into the DB the information for an existing Instance
//	@Tags			Instances
//	@Accept			json
//	@Produce		json
//	@Param			instance	body		inventory.Instance	true	"Instance to be modified"
//	@Success		200			{object}	nil
//	@Failure		500			{object}	nil
//	@Router			/instances/:instance_id [patch]
func HandlerPatchInstance(c *gin.Context) {
	instanceID := c.Param("instance_id")
	logger.Debug("Patching an Instance", zap.String("instance_id", instanceID))
	c.PureJSON(http.StatusNotImplemented, nil)
}

// HandlerGetClusters handles the request for obtaining the entire Cluster list
//	@Summary		Obtain every Cluster
//	@Description	Returns a list of Clusters with a single instance filtered by Name
//	@Tags			Clusters
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	ClusterListResponse
//	@Failure		500	{object}	nil
//	@Router			/clusters/ [get]
func HandlerGetClusters(c *gin.Context) {
	logger.Debug("Retrieving complete clusters inventory")
	addHeaders(c)

	clusters, err := getClusters()
	if err != nil {
		logger.Error("Can't retrieve Clusters list", zap.Error(err))
		c.PureJSON(http.StatusInternalServerError, nil)
		return
	}

	response := NewClusterListResponse(clusters)
	c.PureJSON(http.StatusOK, response)
}

// HandlerGetClustersByName handles the request for obtain an Cluster by its Name
//	@Summary		Obtain a single Cluster by its Name
//	@Description	Returns a list of Clusters with a single Cluster filtered by Name
//	@Tags			Clusters
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	ClusterListResponse
//	@Failure		404	{object}	nil
//	@Failure		500	{object}	nil
//	@Router			/clusters/:cluster_name [get]
func HandlerGetClustersByName(c *gin.Context) {
	clusterName := c.Param("cluster_name")
	logger.Debug("Retrieving Cluster by Name", zap.String("cluster_name", clusterName))
	addHeaders(c)

	clusters, err := getClusterByName(clusterName)
	if err != nil {
		logger.Error("Cluster not found", zap.String("cluster_name", clusterName), zap.Error(err))
		c.PureJSON(http.StatusNotFound, nil)
		return
	}

	response := NewClusterListResponse(clusters)
	c.PureJSON(http.StatusOK, response)
}

// HandlerGetInstancesOnCluster handles the request for obtain the list of Instances belonging to a specific Cluster
//	@Summary		Obtain Instances list belonging to a Cluster
//	@Description	Returns a list of Instances belonging to an Cluster given by Name
//	@Tags			Clusters
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	InstanceListResponse
//	@Failure		500	{object}	nil
//	@Router			/clusters/:cluster_name/instances [get]
func HandlerGetInstancesOnCluster(c *gin.Context) {
	clusterName := c.Param("cluster_name")
	logger.Debug("Retrieving Cluster's Instances", zap.String("cluster_name", clusterName))
	addHeaders(c)

	instances, err := getInstancesOnCluster(clusterName)
	if err != nil {
		logger.Error("Can't retrieve instances on cluster", zap.String("cluster_name", clusterName), zap.Error(err))
		c.PureJSON(http.StatusInternalServerError, nil)
		return
	}

	response := NewInstanceListResponse(instances)
	c.PureJSON(http.StatusOK, response)
}

// HandlerPostCluster handles the request for writting a new Cluster in the inventory
//	@Summary		Creates a new Cluster in the inventory
//	@Description	Receives and write into the DB the information for a new Cluster
//	@Tags			Clusters
//	@Accept			json
//	@Produce		json
//	@Param			cluster	body		inventory.Cluster	true	"New Cluster to be added"
//	@Success		200		{object}	nil
//	@Failure		500		{object}	nil
//	@Router			/clusters/ [post]
func HandlerPostCluster(c *gin.Context) {
	cluster, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		logger.Error("Can't get body from request", zap.Error(err))
		c.PureJSON(http.StatusInternalServerError, nil)
		return
	}
	logger.Debug("Writing a new Cluster", zap.Reflect("cluster", cluster))
	c.PureJSON(http.StatusNotImplemented, nil)
}

// HandlerDeleteCluster handles the request for removing a Cluster in the inventory
//	@Summary		Deletes a Cluster in the inventory
//	@Description	Deletes a Cluster present in the inventory by its Name
//	@Tags			Clusters
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	nil
//	@Failure		500	{object}	nil
//	@Router			/clusters/:cluster_name [delete]
// TODO: Not Implemented
func HandlerDeleteCluster(c *gin.Context) {
	clusterName := c.Param("cluster_name")
	logger.Debug("Removing a Cluster", zap.String("cluster_name", clusterName))
	c.PureJSON(http.StatusNotImplemented, nil)
}

// HandlerPatchCluster handles the request for patching a Cluster in the inventory
//	@Summary		Patches a Cluster in the inventory
//	@Description	Receives and patch into the DB the information for an existing Cluster
//	@Tags			Clusters
//	@Accept			json
//	@Produce		json
//	@Param			instance	body		inventory.Cluster	true	"Cluster to be modified"
//	@Success		200			{object}	nil
//	@Failure		500			{object}	nil
//	@Router			/clusters/:cluster_name [patch]
func HandlerPatchCluster(c *gin.Context) {
	clusterName := c.Param("cluster_name")
	logger.Debug("Patching a Cluster", zap.String("cluster", clusterName))
	c.PureJSON(http.StatusNotImplemented, nil)
}

// HandlerGetAccounts handles the request for obtaining the entire Account list
//	@Summary		Obtain every Account
//	@Description	Returns a list of Accounts with a single Account filtered by Name
//	@Tags			Accounts
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	AccountListResponse
//	@Failure		500	{object}	nil
//	@Router			/accounts/ [get]
func HandlerGetAccounts(c *gin.Context) {
	logger.Debug("Retrieving complete Accounts inventory")
	addHeaders(c)

	accounts, err := getAccounts()
	if err != nil {
		logger.Error("Can't retrieve Accounts list", zap.Error(err))
		c.PureJSON(http.StatusInternalServerError, nil)
		return
	}

	response := NewAccountListResponse(accounts)
	c.PureJSON(http.StatusOK, response)
}

// HandlerGetAccountsByName handles the request for obtain an Account by its Name
//	@Summary		Obtain a single Account by its Name
//	@Description	Returns a list of Accounts with a single Account filtered by Name
//	@Tags			Accounts
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	AccountListResponse
//	@Failure		404	{object}	nil
//	@Failure		500	{object}	nil
//	@Router			/accounts/:account_name [get]
func HandlerGetAccountsByName(c *gin.Context) {
	accountName := c.Param("account_name")
	logger.Debug("Retrieving Account by Name", zap.String("account_name", accountName))
	addHeaders(c)

	accounts, err := getAccountByName(accountName)
	if err != nil {
		logger.Error("Account not found", zap.String("account_name", accountName), zap.Error(err))
		c.PureJSON(http.StatusNotFound, nil)
		return
	}

	response := NewAccountListResponse(accounts)
	c.PureJSON(http.StatusOK, response)
}

// HandlerGetClustersOnAccount handles the request for obtain the list of clusters deployed on a specific Account
//	@Summary		Obtain Cluster list on an Account
//	@Description	Returns a list of Clusters which belongs to an Account given by Name
//	@Tags			Accounts
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	AccountListResponse
//	@Failure		500	{object}	nil
//	@Router			/accounts/:account_name/clusters [get]
func HandlerGetClustersOnAccount(c *gin.Context) {
	accountName := c.Param("account_name")
	logger.Debug("Retrieving Account's Clusters", zap.String("account_name", accountName))
	addHeaders(c)

	clusters, err := getClustersOnAccount(accountName)
	if err != nil {
		logger.Error("Can't retrieve clusters on account", zap.String("account_name", accountName), zap.Error(err))
		c.PureJSON(http.StatusInternalServerError, nil)
		return
	}

	response := NewClusterListResponse(clusters)
	c.PureJSON(http.StatusOK, response)
}

// HandlerPostAccount handles the request for writting a new Account in the inventory
//	@Summary		Creates a new Account in the inventory
//	@Description	Receives and write into the DB the information for a new Account
//	@Tags			Accounts
//	@Accept			json
//	@Produce		json
//	@Param			account	body		inventory.Account	true	"New Account to be added"	Format(email)
//	@Success		200		{object}	nil
//	@Failure		500		{object}	nil
//	@Router			/accounts/ [post]
func HandlerPostAccount(c *gin.Context) {
	account, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		logger.Error("Can't get body from request", zap.Error(err))
		c.PureJSON(http.StatusInternalServerError, nil)
		return
	}
	logger.Debug("Writing a new Account", zap.Reflect("account", account))
	c.PureJSON(http.StatusNotImplemented, nil)
}

// HandlerDeleteAccount handles the request for deleting an Account in the inventory
//	@Summary		Deletes an Account in the inventory
//	@Description	Deletes an Account present in the inventory by its Name
//	@Tags			Accounts
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	nil
//	@Failure		500	{object}	nil
//	@Router			/accounts/:account_name [delete]
// TODO: Not Implemented
func HandlerDeleteAccount(c *gin.Context) {
	accountName := c.Param("account_name")
	logger.Debug("Removing an Account", zap.String("account", accountName))
	c.PureJSON(http.StatusNotImplemented, nil)
}

// HandlerPatchAccount handles the request for patching an Account in the inventory
//	@Summary		Patches an Account in the inventory
//	@Description	Receives and patch into the DB the information for an existing Account
//	@Tags			Accounts
//	@Accept			json
//	@Produce		json
//	@Param			Account	body		inventory.Account	true	"Account to be modified"
//	@Success		200		{object}	nil
//	@Failure		500		{object}	nil
//	@Router			/accounts/:account_name [patch]
func HandlerPatchAccount(c *gin.Context) {
	accountName := c.Param("account_name")
	logger.Debug("Patching an Account", zap.String("account", accountName))
	c.PureJSON(http.StatusNotImplemented, nil)
}
