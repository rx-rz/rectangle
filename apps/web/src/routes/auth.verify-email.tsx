import { createFileRoute } from "@tanstack/react-router";
import { EmailVerification } from "#/features/auth/otp/containers/email-verification";

export const Route = createFileRoute("/auth/verify-email")({
	validateSearch: (search) => ({
		email: typeof search.email === "string" ? search.email : "",
	}),
	component: RouteComponent,
});

function RouteComponent() {
	return <EmailVerification />;
}
