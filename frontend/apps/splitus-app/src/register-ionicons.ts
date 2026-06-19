import { addIcons } from 'ionicons';
import * as ionicons from 'ionicons/icons';

// Register the full ionicons set so any icon referenced by the splitus
// components resolves. Mirrors sneat-app's register-ionicons (which hand-picks
// icons); registering all is simpler for a focused mini-app.
export function registerIonicons(): void {
  const icons = ionicons as unknown as Record<string, string>;
  const map: Record<string, string> = {};
  for (const key of Object.keys(icons)) {
    if (typeof icons[key] === 'string') {
      map[key] = icons[key];
    }
  }
  addIcons(map);
}
