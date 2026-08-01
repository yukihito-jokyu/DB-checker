import { test as base, expect } from "@playwright/test";

import { resetAppEnvironment } from "./app-environment";

export { expect };

export const test = base.extend({
  page: async ({ page }, use) => {
    resetAppEnvironment();
    await use(page);
  },
});
