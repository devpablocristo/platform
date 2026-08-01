package google

import (
	"context"
	"net/http"
)

func (c *Client) QueryFreeBusy(
	ctx context.Context,
	input FreeBusyRequest,
) (FreeBusyResponse, error) {
	if err := required("query freebusy", "TimeMin", input.TimeMin); err != nil {
		return FreeBusyResponse{}, err
	}
	if err := required("query freebusy", "TimeMax", input.TimeMax); err != nil {
		return FreeBusyResponse{}, err
	}
	if len(input.Items) == 0 {
		return FreeBusyResponse{}, validationError("query freebusy", "Items", "must not be empty")
	}
	var output FreeBusyResponse
	_, err := c.doJSON(
		ctx,
		"query freebusy",
		http.MethodPost,
		"/freeBusy",
		nil,
		input,
		"",
		&output,
	)
	if err != nil {
		return FreeBusyResponse{}, err
	}
	return output, nil
}
