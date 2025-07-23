# k8snake

`k8snake` is a command-line interface (CLI) tool that provides a secure and convenient way to interact with services running within a Kubernetes cluster. It handles authentication and connection details, allowing you to focus on the API calls.

The tool is generated from an OpenAPI specification using [oasnake](https://github.com/louislouislouislouis/oasnake) and acts as a client for the defined API.

## Features

-   **Kubernetes Integration:** Connects to your Kubernetes cluster using your local `kubeconfig`.
-   **Automatic Port-Forwarding:** Establishes a port-forwarding session to a specified service and pod within the cluster, making the remote service accessible on your local machine.
-   **Secure Authentication:** Retrieves Keycloak client credentials securely from Kubernetes secrets.
-   **OAuth2/OIDC Integration:** Automatically obtains an authentication token from a Keycloak instance.
-   **Request Modification:** Injects the obtained bearer token and other custom headers into the API requests before forwarding them to the service.

## How it Works

1.  **Initialization:** Reads configuration from the `main.go` file.
2.  **Connect to Kubernetes:** Uses the Go client for Kubernetes to connect to the cluster.
3.  **Find Service:** Locates the specified service and its backing pods in the given namespace.
4.  **Port-Forward:** Sets up a port-forward from a local port to the target service port on one of the pods.
5.  **Fetch Credentials:** Retrieves Keycloak client ID and secret from a specified Kubernetes secret.
6.  **Get Auth Token:** Communicates with the Keycloak token endpoint to get a valid JWT access token.
7.  **Execute Command:** When you run a command to interact with the API, `k8snake` intercepts the request.
8.  **Proxy Request:** It modifies the request to point to the local port-forwarded address and injects the `Authorization: Bearer <token>` header.
9.  **Forward Request:** The modified request is sent to the service running in Kubernetes.

## Prerequisites

-   Go (version 1.24 or higher)
-   A configured `kubectl` with access to a Kubernetes cluster.
-   The target service running in the Kubernetes cluster.
-   A Keycloak instance and a Kubernetes secret containing the client credentials.

## Configuration

Before running the application, you need to update the configuration variables in `main.go`.

```go
// main.go

var (
	/*
	 * Kubernetes configuration
	 */
	k8sNamespace = "your-namespace"      // Namespace of the target service
	serviceName  = "your-service-name"    // Name of the target service
	servicePort  = 8080                   // Port of the target service

	/*
	 * Keycloak Configuration
	 */
	clientSecretID = "your-client-id"     // The Keycloak client ID
	realmName      = "your-realm"         // The Keycloak realm name
	keycloakURL    = "https://your-keycloak/auth/realms/your-realm/protocol/openid-connect/token" // Keycloak token URL

	/*
	 * Custom headers
	 */
	customHeaderName  = "X-Custom-Header"
	customHeaderValue = "some-value"
)
```

## Usage

1.  **Generate API Client:**

    The project uses `go generate` to create the API client code from an OpenAPI specification file (`your-service.yaml`).

    ```sh
    go generate ./...
    ```

2.  **Run the Application:**

    Once the client is generated and the configuration is set, you can run the application.

    ```sh
    go run main.go [command] [flags]
    ```

    The available commands are determined by your OpenAPI specification.

## Build

To build a standalone executable:

```sh
go build -o k8snake
./k8snake [command] [flags]
```