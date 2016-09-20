package policy_client

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"lib/marshal"
	"net/http"

	"code.cloudfoundry.org/lager"
)

type JsonClient struct {
	Logger      lager.Logger
	HttpClient  httpClient
	Url         string
	Marshaler   marshal.Marshaler
	Unmarshaler marshal.Unmarshaler
}

func (c *JsonClient) Do(method, route string, reqData, respData interface{}) error {
	reqURL := c.Url + route
	bodyBytes, err := c.Marshaler.Marshal(reqData)
	if err != nil {
		return fmt.Errorf("json marshal request body: %s", err)
	}
	request, err := http.NewRequest(method, reqURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("http new request: %s", err)
	}
	resp, err := c.HttpClient.Do(request)
	if err != nil {
		return fmt.Errorf("http client do: %s", err)
	}
	defer resp.Body.Close() // untested

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("body read: %s", err)
	}

	if resp.StatusCode > 299 {
		err = fmt.Errorf("http client do: bad response status %d", resp.StatusCode)
		c.Logger.Error("http-client", err, lager.Data{
			"body": string(respBytes),
		})
		return err
	}

	c.Logger.Debug("http-do", lager.Data{
		"body": string(respBytes),
	})

	if respData != nil {
		err = c.Unmarshaler.Unmarshal(respBytes, &respData)
		if err != nil {
			return fmt.Errorf("json unmarshal: %s", err)
		}
	}

	return nil
}
