import { adminAgentSettingsClient } from './connect';
import { create } from '@bufbuild/protobuf';
import {
  GetManagedSettingsRequestSchema,
  SetManagedSettingRequestSchema,
  DeleteManagedSettingRequestSchema,
} from '../gen/ant/v1/admin_settings_pb';
import type { ManagedSettingEntry } from '../gen/ant/v1/admin_settings_pb';

export type { ManagedSettingEntry };

export async function getManagedSettings(): Promise<ManagedSettingEntry[]> {
  const res = await adminAgentSettingsClient.getManagedSettings(
    create(GetManagedSettingsRequestSchema, {}),
  );
  return res.settings;
}

export async function setManagedSetting(key: string, value: string): Promise<boolean> {
  const res = await adminAgentSettingsClient.setManagedSetting(
    create(SetManagedSettingRequestSchema, { key, value }),
  );
  return res.success;
}

export async function deleteManagedSetting(key: string): Promise<boolean> {
  const res = await adminAgentSettingsClient.deleteManagedSetting(
    create(DeleteManagedSettingRequestSchema, { key }),
  );
  return res.success;
}
