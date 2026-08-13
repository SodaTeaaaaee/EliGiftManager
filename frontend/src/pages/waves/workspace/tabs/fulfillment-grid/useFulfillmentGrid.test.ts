import { describe, expect, it } from 'vitest'
import {
  buildShipmentTrackingMap,
  pickPreferredShipmentJoin,
  type ShipmentJoinSource,
} from './useFulfillmentGrid'

const LINE_ID = 10

function shipment(overrides: Partial<ShipmentJoinSource> & Pick<ShipmentJoinSource, 'id'>): ShipmentJoinSource {
  return {
    status: 'shipped',
    trackingNo: 'TRACK-GOOD',
    createdAt: '2026-01-01T00:00:00Z',
    lines: [{ fulfillmentLineId: LINE_ID }],
    ...overrides,
  }
}

function joinFor(shipments: ShipmentJoinSource[]) {
  return buildShipmentTrackingMap(shipments).get(LINE_ID)
}

describe('buildShipmentTrackingMap', () => {
  it('does not let a voided shipment overwrite a non-voided join', () => {
    const live = shipment({ id: 1, status: 'shipped', trackingNo: 'LIVE' })
    const voided = shipment({
      id: 2,
      status: 'voided',
      trackingNo: 'VOIDED',
      createdAt: '2026-06-01T00:00:00Z',
    })
    expect(joinFor([live, voided])).toEqual({ trackingNo: 'LIVE', shipmentTrackingStatus: 'shipped' })
    expect(joinFor([voided, live])).toEqual({ trackingNo: 'LIVE', shipmentTrackingStatus: 'shipped' })
  })

  it('does not let empty tracking overwrite a non-empty tracking number', () => {
    const filled = shipment({ id: 1, trackingNo: 'FILLED', createdAt: '2026-01-01T00:00:00Z' })
    const empty = shipment({ id: 2, trackingNo: '  ', createdAt: '2026-06-01T00:00:00Z' })
    expect(joinFor([filled, empty])).toEqual({ trackingNo: 'FILLED', shipmentTrackingStatus: 'shipped' })
    expect(joinFor([empty, filled])).toEqual({ trackingNo: 'FILLED', shipmentTrackingStatus: 'shipped' })
  })

  it('does not let a later voided shipment with tracking beat an earlier non-voided shipment with tracking', () => {
    const earlierLive = shipment({
      id: 1,
      status: 'shipped',
      trackingNo: 'KEEP',
      createdAt: '2026-01-01T00:00:00Z',
    })
    const laterVoided = shipment({
      id: 99,
      status: 'voided',
      trackingNo: 'VOID-TRACK',
      createdAt: '2026-12-01T00:00:00Z',
    })
    expect(joinFor([earlierLive, laterVoided])).toEqual({ trackingNo: 'KEEP', shipmentTrackingStatus: 'shipped' })
    expect(joinFor([laterVoided, earlierLive])).toEqual({ trackingNo: 'KEEP', shipmentTrackingStatus: 'shipped' })
  })

  it('picks the later createdAt, then higher id, when both joins are non-voided with tracking', () => {
    const earlier = shipment({
      id: 50,
      trackingNo: 'EARLIER',
      createdAt: '2026-01-01T00:00:00Z',
    })
    const later = shipment({
      id: 2,
      trackingNo: 'LATER',
      createdAt: '2026-02-01T00:00:00Z',
    })
    expect(joinFor([later, earlier])).toEqual({ trackingNo: 'LATER', shipmentTrackingStatus: 'shipped' })
    expect(joinFor([earlier, later])).toEqual({ trackingNo: 'LATER', shipmentTrackingStatus: 'shipped' })

    const sameTimeLow = shipment({ id: 3, trackingNo: 'LOW', createdAt: '2026-03-01T00:00:00Z' })
    const sameTimeHigh = shipment({ id: 8, trackingNo: 'HIGH', createdAt: '2026-03-01T00:00:00Z' })
    expect(joinFor([sameTimeHigh, sameTimeLow])).toEqual({ trackingNo: 'HIGH', shipmentTrackingStatus: 'shipped' })
    expect(joinFor([sameTimeLow, sameTimeHigh])).toEqual({ trackingNo: 'HIGH', shipmentTrackingStatus: 'shipped' })
  })
})

describe('pickPreferredShipmentJoin', () => {
  it('prefers non-voided over voided before recency', () => {
    const live = shipment({ id: 1, status: 'in_transit', trackingNo: 'A', createdAt: '2026-01-01T00:00:00Z' })
    const voided = shipment({ id: 9, status: 'voided', trackingNo: 'B', createdAt: '2026-12-01T00:00:00Z' })
    expect(pickPreferredShipmentJoin(live, voided)).toBe(live)
    expect(pickPreferredShipmentJoin(voided, live)).toBe(live)
  })
})
