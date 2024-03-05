package cloudflare

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudflare/cloudflare-go"

	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/transform"
)

type custom_hostname = struct {
	ID        string    `json:"id,omitempty"`
	ZoneID    string    `json:"zone_id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	CreatedOn time.Time `json:"created_on,omitempty"`
}

func tableCloudflareCustomHostname(ctx context.Context) *plugin.Table {
	return &plugin.Table{
		Name:        "cloudflare_custom_hostname",
		Description: "Cloudflare Custom Hostname.",
		List: &plugin.ListConfig{
			KeyColumns: plugin.SingleColumn("zone_id"),
			Hydrate:    listCustomHostname,
		},
		Get: &plugin.GetConfig{
			KeyColumns:        plugin.AllColumns([]string{"zone_id", "id"}),
			ShouldIgnoreError: isNotFoundError([]string{"HTTP status 404"}),
			Hydrate:           getCustomHostname,
		},
		Columns: []*plugin.Column{
			// Top columns
			{Name: "id", Type: proto.ColumnType_STRING, Description: "ID of the custom hostname."},
			{Name: "zone_id", Hydrate: hydrateZoneId, Transform: transform.FromValue(), Type: proto.ColumnType_STRING, Description: "Zone where the custom hostname is defined."},
			{Name: "name", Transform: transform.FromField("Hostname"), Type: proto.ColumnType_STRING, Description: "Custom hostname value."},
			{Name: "status", Type: proto.ColumnType_STRING, Description: "Status of the custom hostname (eg. 'active')."},
			{Name: "created_on", Transform: transform.FromField("CreatedAt"), Type: proto.ColumnType_TIMESTAMP, Description: "When the custom hostname was created."},
			{Name: "ssl", Transform: transform.FromField("SSL").NullIfEmptySlice(), Type: proto.ColumnType_JSON, Description: "SSL meta JSON."},
		},
	}
}

func hydrateZoneId(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	//i := h.Item.(cloudflare.CustomHostname)
	zone := d.EqualsQuals["zone_id"].GetStringValue()
	return zone, nil
}

func listCustomHostname(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (interface{}, error) {
	logger := plugin.Logger(ctx)
	conn, err := connect(ctx, d)
	if err != nil {
		return nil, err
	}
	quals := d.EqualsQuals
	zoneID := quals["zone_id"].GetStringValue()
	ok, exists := quals["name"]
	filter := cloudflare.CustomHostname{}
	if exists {
		filter.Hostname = ok.GetStringValue()
	}

	ok, exists = quals["status"]
	if exists {
		filter.Status = cloudflare.CustomHostnameStatus(ok.GetStringValue())
	}

	page := 1
	for {
		items, result, err := conn.CustomHostnames(ctx, zoneID, page, filter)
		if err != nil {
			logger.Error(fmt.Sprintf("Error found %+v", err))
			return nil, err
		}
		for _, i := range items {
			d.StreamListItem(ctx, i)
		}
		if result.TotalPages <= result.Page || result.Count == 0 {
			break
		}

		page = page + 1
	}

	return nil, nil
}

func getCustomHostname(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (interface{}, error) {
	conn, err := connect(ctx, d)
	if err != nil {
		return nil, err
	}
	quals := d.EqualsQuals
	zoneID := quals["zone_id"].GetStringValue()
	id := quals["id"].GetStringValue()
	item, err := conn.CustomHostname(ctx, zoneID, id)
	if err != nil {
		return nil, err
	}
	return custom_hostname{
		ID:        item.ID,
		ZoneID:    zoneID,
		Name:      item.Hostname,
		Status:    item.SSL.Status,
		CreatedOn: *item.CreatedAt,
	}, nil
}
