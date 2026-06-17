import { IEnvironmentConfig, IFirebaseConfig } from '@sneat/core';

// Production debtus.app config. Reuses the shared sneat production Firebase
// project (sneat-eur3-1) — debtus shares auth, spaces and Firestore with the
// rest of the sneat ecosystem. Swapped in for environment.ts at build time via
// the production fileReplacements in project.json.
const firebaseConfig: IFirebaseConfig = {
  projectId: 'sneat-eur3-1',
  appId: '1:588648831063:web:303af7e0c5f8a7b10d6b12',
  apiKey: 'AIzaSyCeQu1WC182yD0VHrRm4nHUxVf27fY-MLQ',
  // Same-origin authDomain so the signInWithRedirect flow keeps all auth state
  // first-party on debtus.app — the handler, OAuth bounce and getRedirectResult
  // all run on debtus.app, avoiding third-party storage partitioning (which
  // breaks sign-in when authDomain is a different domain like *.firebaseapp.com).
  // Requires https://debtus.app/__/auth/handler on the OAuth client (registered).
  authDomain: 'debtus.app',
  messagingSenderId: '588648831063',
  measurementId: 'G-TYBDTV738R',
};

export const debtusAppEnvironmentConfig: IEnvironmentConfig = {
  production: true,
  agents: {},
  firebaseConfig,
};
