import { appEnvironmentConfig } from '@sneat/app';
import { IEnvironmentConfig } from '@sneat/core';

// Single environment for debtus — fail-safe by construction. appEnvironmentConfig
// returns this production config on every deployed domain and the Firebase
// emulator config only on localhost (decided at runtime from the hostname). No
// environment.prod.ts / fileReplacements: a mis-built or mis-deployed bundle can
// never point real users at the emulator.
//
// Reuses the shared sneat production Firebase project (sneat-eur3-1) — debtus
// shares auth, spaces and Firestore with the rest of the sneat ecosystem.
export const debtusAppEnvironmentConfig: IEnvironmentConfig =
  appEnvironmentConfig({
    production: true,
    agents: {},
    firebaseConfig: {
      projectId: 'sneat-eur3-1',
      appId: '1:588648831063:web:303af7e0c5f8a7b10d6b12',
      apiKey: 'AIzaSyCeQu1WC182yD0VHrRm4nHUxVf27fY-MLQ',
      // Same-origin authDomain so the signInWithRedirect flow keeps all auth
      // state first-party on debtus.app — the handler, OAuth bounce and
      // getRedirectResult all run on debtus.app, avoiding third-party storage
      // partitioning. Requires https://debtus.app/__/auth/handler on the OAuth
      // client (registered).
      authDomain: 'debtus.app',
      messagingSenderId: '588648831063',
      measurementId: 'G-TYBDTV738R',
    },
    // debtus.app is served at its own same-origin authDomain. signInWithPopup is
    // unreliable here under current Chrome COOP behavior, so use a full-page
    // redirect; BaseAppComponent completes it via getRedirectResult().
    signInMethod: 'redirect',
  });
