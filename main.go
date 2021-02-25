package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type ContainerRegistry string

type DockerConfig struct {
	Auths map[ContainerRegistry]Credential `json:"auths"`
}

type Credential struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
	Auth     string `json:"auth"`
}

func main() {
	tokens, err := getECRToken()
	if err != nil {
		panic("Failed to get ECR token: " + err.Error())
	}
	credential, err := generateDockerconfigjson(tokens)
	if err != nil {
		panic("Failed to generate `Dockerconfigjson`: " + err.Error())
	}
	kubeConfig, err := rest.InClusterConfig()
	if err != nil {
		panic("Failed to get cluster config: " + err.Error())
	}
	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		panic("Failed to create clientset: " + err.Error())
	}

	secretName := getEnv("KUBE_SECRET_NAME", "ecr-pull-secret-"+os.Getenv("AWS_REGION"))
	namespace := getEnv("KUBE_NAMESPACE", v1.NamespaceDefault)
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		StringData: map[string]string{
			".dockerconfigjson": string(credential),
		},
		Type: "kubernetes.io/dockerconfigjson",
	}
	result, err := clientset.CoreV1().Secrets(v1.NamespaceDefault).Create(context.TODO(), secret, metav1.CreateOptions{})
	if err != nil {
		panic("Failed to create secret: " + err.Error())
	}
	fmt.Printf("Created secret %q in namespace %q.\n", result.GetObjectMeta().GetName(), result.GetObjectMeta().GetNamespace())
}

// generateDockerconfigjson generates json for data field of `kubernetes.io/dockerconfigjson` type secret
// see: https://kubernetes.io/docs/concepts/configuration/secret/#docker-config-secrets
func generateDockerconfigjson(auths []types.AuthorizationData) ([]byte, error) {
	if auths == nil {
		return nil, errors.New("[]types.AuthorizationData is empty")
	}

	dockerConfig := DockerConfig{Auths: map[ContainerRegistry]Credential{}}
	for _, auth := range auths {
		token := *auth.AuthorizationToken
		decoded, _ := base64.StdEncoding.DecodeString(token)
		substr := strings.Split(string(decoded), ":")
		id := substr[0]
		pw := substr[1]

		dockerConfig.Auths[ContainerRegistry(*auth.ProxyEndpoint)] = Credential{
			Username: id,
			Password: pw,
			Email:    "docker@example.com",
			Auth:     token,
		}
	}

	generated, err := json.Marshal(dockerConfig)
	if err != nil {
		return nil, err
	}
	return generated, nil
}

// getECRToken retrieves auth token(s) for ECR registry using credential via environment variable
func getECRToken() ([]types.AuthorizationData, error) {
	if err := checkEnv("AWS_REGION"); err != nil {
		return nil, err
	}
	if err := checkEnv("AWS_ACCESS_KEY_ID"); err != nil {
		return nil, err
	}
	if err := checkEnv("AWS_SECRET_ACCESS_KEY"); err != nil {
		return nil, err
	}
	fmt.Println("Target region: ", os.Getenv("AWS_REGION"))

	// Assumes AWS token could be retrieved from an environment variable
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, err
	}
	client := ecr.NewFromConfig(cfg)
	response, err := client.GetAuthorizationToken(context.TODO(), nil)
	if err != nil {
		return nil, err
	}
	if response.AuthorizationData == nil {
		return nil, errors.New("there is no ECR account to retrieve token")
	}
	fmt.Println("Retrieved auth count: ", len(response.AuthorizationData))
	fmt.Println("Retrieved auth would expired at ", response.AuthorizationData[0].ExpiresAt)
	for _, auth := range response.AuthorizationData {
		fmt.Println("Registry endpoint: ", *auth.ProxyEndpoint)
	}
	return response.AuthorizationData, nil
}

// checkEnv checks if value of environment variable didn't exist for given key
func checkEnv(key string) error {
	if _, exist := os.LookupEnv(key); !exist {
		return errors.New("Couldn't retrieve environment variable: " + key)
	}
	return nil
}

// getEnv looks up the given key from the environment, returning its value if
// it exists, and otherwise returning the given fallback value.
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
