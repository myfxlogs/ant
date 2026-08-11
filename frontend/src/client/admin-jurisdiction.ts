import { adminJurisdictionClient } from './connect';

export const adminJurisdictionApi = {
  getJurisdictionStatus: async (userId: string) => {
    const resp = await adminJurisdictionClient.getJurisdictionStatus({ userId });
    return resp.status;
  },

  setKYCStatus: async (userId: string, kycStatus: string) => {
    await adminJurisdictionClient.setKYCStatus({ userId, kycStatus });
  },

  listSanctionedCountries: async () => {
    const resp = await adminJurisdictionClient.listSanctionedCountries({});
    return resp.countries ?? [];
  },

  addSanctionedCountry: async (countryCode: string, label: string) => {
    await adminJurisdictionClient.addSanctionedCountry({ countryCode, label });
  },

  removeSanctionedCountry: async (countryCode: string) => {
    await adminJurisdictionClient.removeSanctionedCountry({ countryCode });
  },

  listUsersByKYCStatus: async (params: { kycStatus?: string; page?: number; pageSize?: number }) => {
    const resp = await adminJurisdictionClient.listUsersByKYCStatus({
      kycStatus: params.kycStatus ?? '',
      page: params.page ?? 1,
      pageSize: params.pageSize ?? 20,
    });
    return { users: resp.users ?? [], total: Number(resp.total ?? 0) };
  },

  setSanctionedOverride: async (userId: string, override: boolean) => {
    await adminJurisdictionClient.setSanctionedOverride({ userId, override });
  },
};
