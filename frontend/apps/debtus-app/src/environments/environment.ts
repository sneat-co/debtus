import { appSpecificConfig, emulatorEnvironmentConfig } from '@sneat/app';
import { IEnvironmentConfig } from '@sneat/core';

// Mirrors the sneat-app environment: app-specific config layered over the
// Firebase emulator base. Swap the base for a production config when debtus
// gets its own Firebase project.
export const debtusAppEnvironmentConfig: IEnvironmentConfig =
  appSpecificConfig(emulatorEnvironmentConfig);
