//go:generate oasnake generate --input ./your-service.yaml -o generated  -m github.com/louislouislouislouis/k8snake --with-model --server-url http://localhost:8080 --name k8snake
package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/louislouislouislouis/k8snake/generated/app"
	"github.com/louislouislouislouis/k8snake/generated/app/cmd"
	"github.com/louislouislouislouis/k8snake/generated/app/pkg/config"
	"github.com/louislouislouislouis/k8snake/pkg/services"
	"github.com/louislouislouislouis/k8snake/pkg/utils"
	"github.com/rs/zerolog/log"
)

var (
	/*
	 * Kubernetes configuration
	 */
	k8sNamespace = "TO_COMPLETE"
	serviceName  = "TO_COMPLETE"
	servicePort  = 8080

	/*
	 * Keycloak Configuration
	 */
	clientSecretID = "TO_COMPLETE"
	realmName      = "TO_COMPLETE"
	keycloakURL    = "TO_COMPLETE"
	// keycloakUrl = "https://idm.staging.kiwigrid.com/auth/realms/kiwigrid/protocol/openid-connect/token"
	// keycloakUrl = "https://idm.kiwigrid.com/auth/realms/kiwigrid/protocol/openid-connect/token"

	/*
	 * Custom headers
	 */
	customHeaderName  = "TO_COMPLETE"
	customHeaderValue = "TO_COMPLETE"
)

func main() {
	utils.InitApp()
	log.Debug().Msg("Starting k8snake...")

	/*
	 * token for header based auth
	 */
	var authToken string

	/*
	 * Communication Channel contains infos about the pf services
	 */
	var commChannel services.PFCommunication
	defer func() {
		if commChannel.IsStarted() {
			log.Debug().Msg("Closing communication channel...")
			commChannel.Close()
		}
	}()

	configuration := config.CommandConfig{
		Extensions: config.Extensions{
			RequestModifiers: map[string][]config.RequestModifiers{
				"*": {
					getRequestModifier(&commChannel, &authToken),
				},
			},
			Hooks: map[string][]config.Hook{
				"/things": {
					{
						HookType: config.PersistentPreRun,
						Fns: []func() error{
							getHookFn(&commChannel, &authToken),
						},
					},
				},
			},
		},
	}

	app.Run(cmd.NewK8snakeCmd(configuration))
}

func getRequestModifier(commChannel *services.PFCommunication, authToken *string) func(ctx context.Context, req *http.Request) error {
	return func(ctx context.Context, req *http.Request) error {
		req.Host = fmt.Sprintf("localhost:%d", commChannel.LocalPort)
		req.URL.Host = req.Host
		req.Header.Set("Authorization", "Bearer "+*authToken)
		req.Header.Set("Content-Type", "application/json")
		sub := req.Header.Get(customHeaderName)
		if sub == "" {
			req.Header.Set(customHeaderName, customHeaderValue)
		}
		return nil
	}
}

func getHookFn(commChannel *services.PFCommunication, authToken *string) func() error {
	return func() error {
		k8sSvc, err := services.NewK8sService()
		if err != nil {
			return err
		}

		// TODO: Use a more appropriate context
		ctx := context.TODO()

		svc, err := k8sSvc.GetService(k8sNamespace, serviceName, ctx)
		if err != nil {
			return err
		}

		pods, err := k8sSvc.GetPodsForSvc(svc, k8sNamespace, ctx)
		if err != nil {
			return err
		}
		if len(pods.Items) == 0 {
			return fmt.Errorf("no pods found for service %s in namespace %s", serviceName, k8sNamespace)
		}

		newChannel, err := k8sSvc.PortForwardSVC(k8sNamespace, pods.Items[0].Name, servicePort)
		if err != nil {
			return err
		}
		// Overwrite the communication channel value of the pointer with the new one
		*commChannel = *newChannel
		secName := fmt.Sprintf("keycloak-client-secret-%s-%s", clientSecretID, realmName)

		sec, err := k8sSvc.GetSecret(k8sNamespace, secName, ctx)
		if err != nil {
			panic(fmt.Sprintf("failed to get secret %s in namespace %s: %v", secName, k8sNamespace, err))
		}

		keycloakTokenManager := services.NewKeycloakTokenManager(
			services.KeycloakConfig{
				Credentials: services.Credentials{
					ClientID:     string(sec.Data["CLIENT_ID"]),
					ClientSecret: string(sec.Data["CLIENT_SECRET"]),
				},
				TokenURL: keycloakURL,
			},
		)

		tokenVal, err := keycloakTokenManager.GetValidToken()
		if err != nil {
			return err
		}

		// Overwrite the value of the string pointer with the new one
		*authToken = tokenVal
		return nil
	}
}
