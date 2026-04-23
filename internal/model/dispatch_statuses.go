package model

// DispatchStatusPending 表示记录已进入批次，但还未进入后续发货处理。
const DispatchStatusPending = "pending"

// DispatchStatusPendingAddress 表示记录缺少可用地址，暂时不能继续导出。
const DispatchStatusPendingAddress = "pending_address"
