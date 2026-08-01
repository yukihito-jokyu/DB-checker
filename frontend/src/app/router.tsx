import { createHashRouter } from "react-router-dom";

import App from "@/App";
import { ConnectionProfilesPage } from "@/pages/connection-profiles/ConnectionProfilesPage";

export const router = createHashRouter([
  {
    path: "/",
    element: <App />,
    children: [
      {
        index: true,
        element: <ConnectionProfilesPage />,
      },
    ],
  },
]);
