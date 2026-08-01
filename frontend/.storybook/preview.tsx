import type { Preview } from "@storybook/react-vite";
import { sb } from "storybook/test";

import "../src/index.css";

sb.mock(
  import("../src/features/connection-profiles/services/connectionProfiles.ts"),
);

const preview: Preview = {
  decorators: [
    (Story) => {
      document.documentElement.lang = "ja";
      return <Story />;
    },
  ],
  parameters: {
    layout: "padded",
    controls: {
      matchers: {
        color: /(background|color)$/i,
        date: /Date$/i,
      },
    },
    a11y: {
      test: "error",
    },
  },
  tags: ["autodocs"],
};

export default preview;
