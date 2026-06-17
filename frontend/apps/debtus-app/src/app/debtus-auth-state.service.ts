import { Injectable } from '@angular/core';
import {
  AuthProvider,
  FacebookAuthProvider,
  getRedirectResult,
  GithubAuthProvider,
  GoogleAuthProvider,
  OAuthProvider,
  signInWithRedirect,
  UserCredential,
} from '@angular/fire/auth';
import { Capacitor } from '@capacitor/core';
import { AuthProviderID, SneatAuthStateService } from '@sneat/auth-core';

function getAuthProvider(authProviderID: AuthProviderID): AuthProvider {
  switch (authProviderID) {
    case 'google.com':
      return new GoogleAuthProvider();
    case 'apple.com':
      return new OAuthProvider('apple.com');
    case 'microsoft.com':
      return new OAuthProvider('microsoft.com');
    case 'facebook.com': {
      const facebookAuthProvider = new FacebookAuthProvider();
      facebookAuthProvider.addScope('email');
      return facebookAuthProvider;
    }
    case 'github.com': {
      const githubAuthProvider = new GithubAuthProvider();
      githubAuthProvider.addScope('read:user');
      githubAuthProvider.addScope('user:email');
      return githubAuthProvider;
    }
    default:
      throw new Error('unsupported auth provider: ' + authProviderID);
  }
}

// debtus.app runs on its own domain with a same-origin authDomain. Google's
// OAuth popup pages set a Cross-Origin-Opener-Policy that severs the
// opener<->popup link, so signInWithPopup hangs (the popup closes but its result
// never reaches the app). Redirect avoids popups entirely and is reliable now
// that authDomain is same-origin (no cross-domain storage partitioning issue).
// The redirect result is picked up on return via the base service's
// onIdTokenChanged/onAuthStateChanged handlers, and the login page then routes
// the user onward. Native builds keep the base (native-layer) flow.
@Injectable()
export class DebtusAuthStateService extends SneatAuthStateService {
  constructor() {
    super();
    // Complete any pending signInWithRedirect on app load. The base service's
    // onAuthStateChanged then picks up the signed-in user. Harmless (returns
    // null) when there is no pending redirect.
    getRedirectResult(this.fbAuth).catch((err) =>
      console.error('getRedirectResult failed', err),
    );
  }

  public override async signInWith(
    authProviderID: AuthProviderID,
  ): Promise<UserCredential | undefined> {
    if (Capacitor.isNativePlatform()) {
      return super.signInWith(authProviderID);
    }
    const authProvider = getAuthProvider(authProviderID);
    await signInWithRedirect(this.fbAuth, authProvider);
    return undefined;
  }
}
