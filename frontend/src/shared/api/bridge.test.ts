// @vitest-environment happy-dom

// Behavior pins for the Wails v3 bridge (shared/api/bridge.ts):
//   - degradation: without a wails host (happy-dom has no window.chrome
//     webview / webkit / wails bridge objects) list-style getters resolve
//     empty containers, settings/home getters resolve documented default
//     objects, pickFile resolves null, and mutating commands reject with
//     'Wails runtime is not available';
//   - guard side effects: one guard evaluation flips the shared bridge
//     health state (markBridgeMissing / markBridgeSeen via useBridgeHealth);
//   - wire handling: what the bridge rebuilds before the UI sees it
//     (inspectSampleFile) versus what it deliberately passes through
//     untouched (listResultViews, getHomeBuckets), where the Go side carries
//     the non-null guarantee instead;
//   - cancel contract: pickFile maps both user-cancel rejection shapes to
//     null — the wails wrapper prefix wrapping a zh-CN OS cancel message
//     (no "cancel" substring to match) and the runtime's English
//     "canceled" message — and rethrows genuine failures untouched.
//
// The generated controller modules and the @wailsio/runtime package are
// mocked wholesale (hoisted mock records + vi.mock factories), so no real
// Wails machinery ever loads; the runtime mock is what lets the
// cancel-contract cases drive Dialogs.OpenFile rejections. Mocking the
// runtime package whole (rather than spying on the real module) is
// deliberate: `Dialogs` is an ES module namespace object whose exports are
// non-writable, so a vi.spyOn could throw. This file never imports
// frontend/bindings directly — that stays bridge.ts's exclusive privilege
// (vi.mock specifiers are function arguments, not import statements).

import { beforeEach, describe, expect, it, vi, type Mock } from 'vitest'

import {
  createPlatform,
  getHomeBuckets,
  getSettings,
  inspectSampleFile,
  listPlatforms,
  listResultViews,
  listWaves,
  pickFile,
} from './bridge'
import { useBridgeHealth } from './health'

// The exact specifiers bridge.ts uses, resolved from this directory, so the
// mocks intercept precisely the modules the bridge imports. Every name the
// bridge binds must exist on the mock module, otherwise the bridge import
// itself would fail on a missing export.
const controllerMocks = vi.hoisted(() => {
  const toMocks = (names: string[]): Record<string, Mock> => {
    const mocks: Record<string, Mock> = {}
    for (const name of names) mocks[name] = vi.fn()
    return mocks
  }
  return {
    workspace: toMocks([
      'ApplyRevision', 'AssignLines', 'AttachIdentity', 'CloseWave',
      'DecideDuplicate', 'DismissRevision', 'MoveLines', 'CreateAddress',
      'CreateAlias', 'CreateBundleComponent', 'CreateCarrierMapping',
      'CreateCustomer', 'CreateGrant', 'CreatePlatform', 'CreateProduct',
      'CreateTemplate', 'CreateWave', 'AddException', 'DeleteCarrierMapping',
      'DeleteException', 'DeleteQuantitySplitRule', 'DeleteRule',
      'DeleteTemplate', 'DocumentTypeCatalog', 'ListQuantitySplitRules',
      'UpsertQuantitySplitRule', 'ExportFactoryOrderFile',
      'ExportWritebackFile', 'GenerateFactoryOrder',
      'GenerateFactoryOrderForResults', 'GenerateWritebacks', 'GetCustomer',
      'GetSettings', 'GetTemplate', 'GetWave', 'Home', 'ImportCarrierMappings',
      'ImportFile', 'ImportShipment', 'ImportShipmentFile', 'IngestDocument',
      'InspectSampleFile', 'ListAddresses', 'ListAliases',
      'ListCarrierMappings', 'ListCustomers', 'ListEntitlementInstances',
      'ListExceptions', 'ListInboxRows', 'ListPlatforms', 'ListProducts',
      'ListResultViews', 'ListRules', 'ListSupplierOrderLines',
      'ListSupplierOrders', 'ListTemplates', 'ListWaves',
      'ListWritebacksByWave', 'MarkWritebackFailed', 'MarkWritebackSent',
      'NamedTransformers', 'ProductTotals', 'PreviewMapping',
      'PreviewTemplate', 'ReopenWave', 'SaveSettings', 'SemanticDictionary',
      'SetResultAddress', 'UpdateAlias', 'UpdateCarrierMapping',
      'UpdateCustomer', 'UpdateTemplate', 'UpsertRule', 'VoidFactoryOrder',
    ]),
    filesystem: toMocks(['GetDataDir', 'RevealInFolder']),
  }
})

vi.mock(
  '../../../bindings/github.com/SodaTeaaaaee/EliGiftManager/internal/controller/workspacecontroller',
  () => controllerMocks.workspace,
)

vi.mock(
  '../../../bindings/github.com/SodaTeaaaaee/EliGiftManager/internal/controller/filesystemcontroller',
  () => controllerMocks.filesystem,
)

// bridge.ts binds `import { Dialogs } from '@wailsio/runtime'` by name, so
// the factory below only has to provide the Dialogs namespace with an
// OpenFile spy for the pickFile cancel-contract cases.
const runtimeMocks = vi.hoisted(() => {
  const dialogs: Record<string, Mock> = { OpenFile: vi.fn() }
  return { dialogs }
})

vi.mock('@wailsio/runtime', () => ({ Dialogs: runtimeMocks.dialogs }))

// happy-dom provides a DOM but no wails host objects, which is precisely the
// degraded environment the bridge guard must handle. Installing a fake
// WebView2-style host flips the guard open so the pass-through paths below
// can consume the mocked controller responses.
function installFakeWailsHost(): void {
  const w = window as unknown as { chrome?: unknown }
  w.chrome = { webview: { postMessage: () => {} } }
}

function removeFakeWailsHost(): void {
  delete (window as unknown as { chrome?: unknown }).chrome
}

beforeEach(() => {
  removeFakeWailsHost()
  vi.resetAllMocks()
})

describe('bridge degradation without a wails host', () => {
  it('flips the shared bridge health state from unknown to unavailable after one guard miss', async () => {
    expect(useBridgeHealth().bridgeState.value).toBe('unknown')

    await listPlatforms()

    expect(useBridgeHealth().bridgeState.value).toBe('unavailable')
  })

  it('list-style getters degrade to [] without consuming the generated bindings', async () => {
    await expect(listPlatforms()).resolves.toEqual([])
    await expect(listWaves()).resolves.toEqual([])
    await expect(listResultViews(1)).resolves.toEqual([])

    expect(controllerMocks.workspace['ListPlatforms']).not.toHaveBeenCalled()
    expect(controllerMocks.workspace['ListWaves']).not.toHaveBeenCalled()
    expect(controllerMocks.workspace['ListResultViews']).not.toHaveBeenCalled()
  })

  it('getSettings() resolves the five-field default object', async () => {
    await expect(getSettings()).resolves.toEqual({
      Locale: 'zh-CN',
      Theme: 'system',
      Density: 'comfortable',
      DuplicateRecordMinutes: 10,
      DuplicateAskDays: 10,
    })
    expect(controllerMocks.workspace['GetSettings']).not.toHaveBeenCalled()
  })

  it('getHomeBuckets() resolves the ten-field default object', async () => {
    await expect(getHomeBuckets()).resolves.toEqual({
      Unassigned: 0,
      DuplicateAsk: 0,
      AlignmentConflict: 0,
      IdentityUnattached: 0,
      PendingRevisions: 0,
      BlockedResults: 0,
      WritebackFailed: 0,
      ResidualClose: 0,
      RevisionFrozenConflicts: 0,
      RecentWaves: [],
    })
    expect(controllerMocks.workspace['Home']).not.toHaveBeenCalled()
  })

  it('pickFile() resolves null without touching the runtime dialogs API', async () => {
    await expect(pickFile()).resolves.toBeNull()
    expect(runtimeMocks.dialogs['OpenFile']).not.toHaveBeenCalled()
  })

  it('createPlatform() rejects with the guard error before reaching the binding', async () => {
    await expect(createPlatform({})).rejects.toThrow('Wails runtime is not available')
    expect(controllerMocks.workspace['CreatePlatform']).not.toHaveBeenCalled()
  })
})

describe('bridge pass-through with a wails host', () => {
  beforeEach(() => {
    installFakeWailsHost()
  })

  it('marks the shared bridge health state available on the first guarded call', async () => {
    await listPlatforms()

    expect(useBridgeHealth().bridgeState.value).toBe('available')
  })

  it('listResultViews passes the wire array through untouched — Blocks: null stays null', async () => {
    // Reality pin, not a normalization promise: listResultViews returns the
    // wire array as-is (a double cast, no rebuild), so a legacy wire shape
    // with `Blocks: null` surfaces null to callers. The non-null contract
    // the UI relies on is carried by the Go side ("the backend always sends
    // a Blocks array", see ResultView in @/entities/models); this test
    // deliberately does not paper over that with a bridge-side fallback.
    const wire = [
      { Result: { ID: 1 }, WorkState: 'ready', Blocks: null, InFactory: false, Shipped: false, WritebackFailed: false },
      { Result: { ID: 2 }, WorkState: 'blocked', Blocks: ['unaligned_product'], InFactory: false, Shipped: false, WritebackFailed: false },
    ]
    controllerMocks.workspace['ListResultViews'].mockResolvedValue(wire)

    await expect(listResultViews(7)).resolves.toBe(wire)
    expect(controllerMocks.workspace['ListResultViews']).toHaveBeenCalledWith(7)
  })

  it('getHomeBuckets passes the wire object through untouched — RecentWaves: null stays null', async () => {
    // Same reality pin: getHomeBuckets casts the Home() result as-is, so a
    // null RecentWaves surfaces null. The facade types it non-null because
    // the Go side always materializes [] ("backend sends []", see
    // HomeBuckets in @/entities/models) — Go, not the bridge, owns that
    // guarantee.
    const wire = {
      Unassigned: 3,
      DuplicateAsk: 1,
      AlignmentConflict: 0,
      IdentityUnattached: 2,
      PendingRevisions: 0,
      BlockedResults: 4,
      WritebackFailed: 0,
      ResidualClose: 1,
      RevisionFrozenConflicts: 0,
      RecentWaves: null,
    }
    controllerMocks.workspace['Home'].mockResolvedValue(wire)

    await expect(getHomeBuckets()).resolves.toBe(wire)
  })

  it('inspectSampleFile rebuilds the wire: null sheets/records/rows/cells become [] and ""', async () => {
    // The one true bridge-side normalization: the defensive mapping mirrors
    // the v2 baseline so a hostile or legacy wire shape never leaks nulls
    // into the template editor.
    controllerMocks.workspace['InspectSampleFile'].mockResolvedValue({
      Format: 'csv',
      Sheets: null,
      Records: [['order_no', null], null],
      Total: 5,
    })

    await expect(inspectSampleFile('sample.csv', '', 10)).resolves.toEqual({
      Format: 'csv',
      Sheets: [],
      Records: [['order_no', ''], []],
      Total: 5,
    })
    expect(controllerMocks.workspace['InspectSampleFile']).toHaveBeenCalledWith('sample.csv', '', 10)
  })

  it('pickFile maps a zh-CN OS cancel (wrapper prefix + localized text) to null', async () => {
    // Windows describes the cancel HRESULT in the system locale, so the
    // message carries no "cancel" substring; only the locale-free wails
    // wrapper prefix can match here.
    runtimeMocks.dialogs['OpenFile'].mockRejectedValueOnce(
      new Error('Invalid dialog call: Dialog.OpenFile failed, error getting selection: 操作已被用户取消。'),
    )

    await expect(pickFile()).resolves.toBeNull()
  })

  it('pickFile maps the runtime English cancel message to null', async () => {
    runtimeMocks.dialogs['OpenFile'].mockRejectedValueOnce(
      new Error('The operation was canceled by the user.'),
    )

    await expect(pickFile()).resolves.toBeNull()
  })

  it('pickFile rethrows a genuine dialog failure untouched', async () => {
    runtimeMocks.dialogs['OpenFile'].mockRejectedValueOnce(
      new Error('some real failure'),
    )

    await expect(pickFile()).rejects.toThrow('some real failure')
  })
})
