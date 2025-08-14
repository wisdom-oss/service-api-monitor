package healthchecks

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"

	"microservice/internal/configuration"
)

// Base is a very basic healthcheck that pings the database server and returns an
// error if the connection could not be established.
func Base(ctx context.Context) error {
	c := configuration.Default.Viper()
	baseAPI := c.GetString(configuration.ConfigurationKey_TraefikAPIEndpoint)

	uri, err := url.JoinPath(baseAPI, "/api", "/overview")
	if err != nil {
		return err
	}

	res, err := http.Get(uri) //nolint:gosec
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return errors.New("api gateway responded with not-OK (status != 200) to overview request")
	}

	if _, err := os.Open(".server-running"); os.IsNotExist(err) {
		return nil
	}

	isPlain, ok := ctx.Value("plain").(bool)
	if !ok || !isPlain {
		return nil
	}

	host := net.JoinHostPort("localhost", c.GetString(configuration.ConfigurationKey_HttpPort))

	res, err = http.Get("http://" + host + "/_/health")
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("own http server responded with not-OK (status = %d)", res.StatusCode)
	}

	return nil

}
