package collab

import (
	"fmt"

	"github.com/reearth/reearth/server/pkg/id"
	"github.com/reearth/reearth/server/pkg/nlslayer"
)

// geometryToGeoJSONMap builds a GeoJSON-style object with a "coordinates" array
// suitable for nlslayer.NewGeometryFromMap (same shape as collab / GraphQL JSON input).
func geometryToGeoJSONMap(g nlslayer.Geometry) (map[string]any, error) {
	switch x := g.(type) {
	case *nlslayer.Point:
		coords := x.Coordinates()
		arr := make([]any, len(coords))
		for i, c := range coords {
			arr[i] = c
		}
		return map[string]any{"type": x.PointType(), "coordinates": arr}, nil
	case *nlslayer.LineString:
		coords := x.Coordinates()
		outer := make([]any, len(coords))
		for i, pt := range coords {
			inner := make([]any, len(pt))
			for j, c := range pt {
				inner[j] = c
			}
			outer[i] = inner
		}
		return map[string]any{"type": x.LineStringType(), "coordinates": outer}, nil
	case *nlslayer.Polygon:
		coords := x.Coordinates()
		outer := make([]any, len(coords))
		for i, ring := range coords {
			ringArr := make([]any, len(ring))
			for j, pt := range ring {
				ptArr := make([]any, len(pt))
				for k, c := range pt {
					ptArr[k] = c
				}
				ringArr[j] = ptArr
			}
			outer[i] = ringArr
		}
		return map[string]any{"type": x.PolygonType(), "coordinates": outer}, nil
	case *nlslayer.MultiPolygon:
		coords := x.Coordinates()
		outer := make([]any, len(coords))
		for i, poly := range coords {
			polyArr := make([]any, len(poly))
			for j, ring := range poly {
				ringArr := make([]any, len(ring))
				for k, pt := range ring {
					ptArr := make([]any, len(pt))
					for m, c := range pt {
						ptArr[m] = c
					}
					ringArr[k] = ptArr
				}
				polyArr[j] = ringArr
			}
			outer[i] = polyArr
		}
		return map[string]any{"type": x.MultiPolygonType(), "coordinates": outer}, nil
	case *nlslayer.GeometryCollection:
		geoms := x.Geometries()
		list := make([]any, 0, len(geoms))
		for _, sub := range geoms {
			m, err := geometryToGeoJSONMap(sub)
			if err != nil {
				return nil, err
			}
			list = append(list, m)
		}
		return map[string]any{"type": x.GeometryCollectionType(), "geometries": list}, nil
	default:
		return nil, fmt.Errorf("unsupported geometry type %T", g)
	}
}

func findNLSFeature(layer nlslayer.NLSLayer, fid id.FeatureID) (*nlslayer.Feature, error) {
	if layer.Sketch() == nil || layer.Sketch().FeatureCollection() == nil {
		return nil, fmt.Errorf("sketch or feature collection missing")
	}
	feats := layer.Sketch().FeatureCollection().Features()
	for i := range feats {
		if feats[i].ID() == fid {
			cp := feats[i]
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("feature not found")
}
