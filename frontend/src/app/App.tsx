import { RouterProvider } from "react-router-dom";

import { ToastViewport } from "@/components/ToastViewport";
import { router } from "@/router";

export function App() {
  return (
    <>
      <RouterProvider router={router} />
      <ToastViewport />
    </>
  );
}
