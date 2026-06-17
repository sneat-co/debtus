import { IEnvironmentConfig, IFirebaseConfig } from '@sneat/core';

// Production debtus.app config. Reuses the shared sneat production Firebase
// project (sneat-eur3-1) — debtus shares auth, spaces and Firestore with the
// rest of the sneat ecosystem. Swapped in for environment.ts at build time via
// the production fileReplacements in project.json.
const firebaseConfig: IFirebaseConfig = {
  projectId: 'sneat-eur3-1',
  appId: '1:588648831063:web:303af7e0c5f8a7b10d6b12',
  apiKey: 'AIzaSyCeQu1WC182yD0VHrRm4nHUxVf27fY-MLQ',
  // Use the real Firebase-managed handler domain. The custom-domain
  // (debtus.app) /__/auth/handler proxies the default config and supports the
  // popup bounce but NOT the top-level signInWithRedirect leg ("requested action
  // is invalid"). The genuine handler at *.firebaseapp.com fully supports
  // redirect, and its redirect URI is already registered on the OAuth client.
  authDomain: 'sneat-eur3-1.firebaseapp.com',
  messagingSenderId: '588648831063',
  measurementId: 'G-TYBDTV738R',
};

export const debtusAppEnvironmentConfig: IEnvironmentConfig = {
  production: true,
  agents: {},
  firebaseConfig,
};
