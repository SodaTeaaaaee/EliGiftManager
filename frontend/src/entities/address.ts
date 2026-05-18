export interface CustomerAddressDTO {
  id: number
  customerProfileId: number
  label: string
  recipientName: string
  phone: string
  country: string
  province: string
  city: string
  district: string
  addressLine1: string
  addressLine2: string
  postalCode: string
  isDefault: boolean
  isTest: boolean
  validationStatus: string
  validationDetail: string
  extraData: string
  createdAt: string
  updatedAt: string
}

export interface CreateAddressInput {
  customerProfileId: number
  label: string
  recipientName: string
  phone: string
  country: string
  province: string
  city: string
  district: string
  addressLine1: string
  addressLine2: string
  postalCode: string
  isDefault: boolean
  isTest: boolean
  validationStatus: string
  validationDetail: string
  extraData: string
}

export interface UpdateAddressInput {
  id: number
  customerProfileId: number
  label: string
  recipientName: string
  phone: string
  country: string
  province: string
  city: string
  district: string
  addressLine1: string
  addressLine2: string
  postalCode: string
  isDefault: boolean
  isTest: boolean
  validationStatus: string
  validationDetail: string
  extraData: string
}

export interface BindAddressInput {
  fulfillmentLineId: number
  customerAddressId: number
}
