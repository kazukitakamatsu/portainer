package intel

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	httperror "github.com/portainer/libhttp/error"
	"github.com/portainer/libhttp/request"
	"github.com/portainer/libhttp/response"
	portainer "github.com/portainer/portainer/api"
	bolterrors "github.com/portainer/portainer/api/bolt/errors"
	"github.com/sirupsen/logrus"
)

type Info struct {
	Endpoint portainer.EndpointID
	Text     string
}

// @id OpenAMTHostInfo
// @summary Request OpenAMT info from a node
// @description Request OpenAMT info from a node
// @description **Access policy**: administrator
// @tags intel
// @security jwt
// @accept json
// @produce json
// @success 204 "Success"
// @failure 400 "Invalid request"
// @failure 403 "Permission denied to access settings"
// @failure 500 "Server error"
// @router /manage/{envid}/info [post]
func (handler *Handler) OpenAMTHostInfo(w http.ResponseWriter, r *http.Request) *httperror.HandlerError {
	endpointID, err := request.RetrieveNumericRouteVariableValue(r, "id")
	if err != nil {
		return &httperror.HandlerError{http.StatusBadRequest, "Invalid environment identifier route variable", err}
	}

	logrus.WithField("endpointID", endpointID).Info("OpenAMTHostInfo")

	endpoint, err := handler.DataStore.Endpoint().Endpoint(portainer.EndpointID(endpointID))
	if err == bolterrors.ErrObjectNotFound {
		return &httperror.HandlerError{StatusCode: http.StatusNotFound, Message: "Unable to find an endpoint with the specified identifier inside the database", Err: err}
	} else if err != nil {
		return &httperror.HandlerError{StatusCode: http.StatusInternalServerError, Message: "Unable to find an endpoint with the specified identifier inside the database", Err: err}
	}

	// TODO: start
	//      docker run --rm -it --privileged ptrrd/openamt:rpc-go amtinfo
	//		on the Docker standalone node (one per env :)
	//		and later, on the specified node in the swarm, or kube.
	nodeName := ""
	docker, err := handler.DockerClientFactory.CreateClient(endpoint, nodeName)
	if err != nil {
		return &httperror.HandlerError{StatusCode: http.StatusInternalServerError, Message: "Unable to find an endpoint with the specified identifier inside the database", Err: err}
	}
	defer docker.Close()

	ctx := context.TODO()
	// pull the image so we can check if there's a new one
	imagename := "ptrrd/openamt:rpc-go"
	containerName := "openamt-rpc-go"

	if err := pullImage(ctx, docker, imagename); err != nil {
		return &httperror.HandlerError{StatusCode: http.StatusInternalServerError,
			Message: "Could not pull image from registry", Err: err}
	}

	output, err := runContainer(ctx, docker, imagename, containerName, []string{"amtinfo"})
	if err != nil {
		return &httperror.HandlerError{StatusCode: http.StatusInternalServerError,
			Message: "Could not run container", Err: err}
	}

	amtInfo := Info{
		Endpoint: portainer.EndpointID(endpointID),
		Text:     output,
	}
	return response.JSON(w, amtInfo)
}

// TODO: ideally, pullImage and runContainer will become a simple version of the use compose abstraction that can be called from withing Portainer.
// TODO: the idea being that if we have an internal struct of a parsed compose file, we can also populate that struct programatically, and run it to get the result I'm getting here.
// TODO: likley an upgrade and absrtaction of DeployComposeStack/DeploySwarmStack/DeployKubernetesStack
// pullImage will pull the image to the specified environment
// TODO: add k8s implemenation
// TODO: work out registry auth
func pullImage(ctx context.Context, docker *client.Client, imagename string) error {
	r, err := docker.ImagePull(ctx, imagename, types.ImagePullOptions{})
	if err != nil {
		logrus.WithError(err).Error("Could not pull %s from registry", imagename)
		return err
	}
	// yeah, swiped this, need to figure out a good way to wait til its done...
	b := make([]byte, 8)
	for {
		_, err := r.Read(b)
		// TODO: should convert json text to a struct and show just the text messages
		//if n > 0 {
		//fmt.Printf(string(b))
		//}
		if err == io.EOF {
			break
		}
	}
	r.Close()

	return nil
}

// TODO: ideally, pullImage and runContainer will become a simple version of the use compose abstraction that can be called from withing Portainer.
// runContainer should be used to run a short command that returns information to stdout
// TODO: add k8s support
func runContainer(ctx context.Context, docker *client.Client, imagename, containerName string, cmd []string) (output string, err error) {
	envs := []string{}
	// for _, envKey := range envKeys {
	// 	envs = append(envs, options.GetConfigAsEnv(envKey))
	// }
	create, err := docker.ContainerCreate(
		ctx,
		&container.Config{
			Image: imagename,
			Cmd:   cmd,
			Env:   envs,
			// ExposedPorts: nat.PortSet{
			// 	nat.Port("80/tcp"):   {},
			// 	nat.Port("443/tcp"):  {},
			// 	nat.Port("2019/tcp"): {},
			// },
			Tty:          true,
			OpenStdin:    true,
			AttachStdout: true,
			AttachStderr: true,
		},
		&container.HostConfig{
			Privileged: true,
			// RestartPolicy: container.RestartPolicy{
			// 	Name: "unless-stopped",
			// },
			// Mounts: []mount.Mount{
			// 	mount.Mount{
			// 		Type:   mount.TypeBind,
			// 		Source: "/var/run/docker.sock",
			// 		Target: "/var/run/docker.sock",
			// 	},
			// 	mount.Mount{
			// 		Type:   mount.TypeVolume,
			// 		Source: "cirri_caddy_data",
			// 		Target: "/data",
			// 	},
			// 	mount.Mount{
			// 		Type:   mount.TypeVolume,
			// 		Source: "cirri_caddy_auth",
			// 		Target: "/config/caddy/localauth",
			// 	},
			// 	mount.Mount{
			// 		Type:   mount.TypeVolume,
			// 		Source: "cirri_caddy_rolemapping",
			// 		Target: "/config/caddy/rolemapping",
			// 	},
			// },
			// PortBindings: nat.PortMap{
			// 	nat.Port("80/tcp"):  []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "80"}},
			// 	nat.Port("443/tcp"): []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "443"}},
			// },
			//NetworkMode: "cirri_proxy",
		},
		&network.NetworkingConfig{},
		nil,
		containerName)
	if err != nil {
		fmt.Printf("ERROR creating container: %s\n", err)
		panic(err)
	}
	err = docker.ContainerStart(ctx, create.ID, types.ContainerStartOptions{})
	if err != nil {
		logrus.WithError(err).WithField("imagename", imagename).WithField("containername", containerName).Error("starting container")
		return "", err
	}

	log.Printf("%s container created and started\n", containerName)

	statusCh, errCh := docker.ContainerWait(ctx, create.ID, container.WaitConditionNotRunning)
	var statusCode int64
	select {
	case err := <-errCh:
		if err != nil {
			logrus.WithError(err).WithField("imagename", imagename).WithField("containername", containerName).Error("starting container")
			return "", err
		}
	case status := <-statusCh:
		statusCode = status.StatusCode
	}
	fmt.Printf("STATUS: %v\n", statusCode)

	out, err := docker.ContainerLogs(ctx, create.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		logrus.WithError(err).WithField("imagename", imagename).WithField("containername", containerName).Error("getting container log")
		return "", err
	}

	err = docker.ContainerRemove(ctx, create.ID, types.ContainerRemoveOptions{})
	if err != nil {
		logrus.WithError(err).WithField("imagename", imagename).WithField("containername", containerName).Error("removing container")
		return "", err
	}

	outputBytes, err := ioutil.ReadAll(out)
	if err != nil {
		logrus.WithError(err).WithField("imagename", imagename).WithField("containername", containerName).Error("read container output")
		return "", err
	}
	return string(outputBytes), nil
}
