import { createFileRoute } from "@tanstack/react-router";
import { LoginForm } from "#/features/auth/login/containers/login";
import { OauthContainer } from "#/features/auth/signup/containers/oauth";

export const Route = createFileRoute("/auth/login")({
	validateSearch: (search: Record<string, unknown>) => ({
		oauth_error:
			typeof search.oauth_error === "string" ? search.oauth_error : undefined,
	}),
	component: RouteComponent,
});

function RouteComponent() {
	const { oauth_error: oauthError } = Route.useSearch();
	const oauthErrorMessage = getOauthErrorMessage(oauthError);

	return <div className='flex flex-col gap-6'>
		<LoginForm />
		{oauthErrorMessage ? (
			<p className="rounded-md border  border-destructive/30 bg-destructive/10 px-3 py-2 text-destructive text-sm">
				{oauthErrorMessage}
			</p>
		) : null}

		<div className='flex items-center gap-4 text-sm'>
			<div className='bg-muted h-px flex-1'></div>
			OR
			<div className='bg-muted h-px flex-1 '></div>
		</div>
		<OauthContainer />
	</div>
}

function getOauthErrorMessage(error?: string) {
	if (error === "use_existing_login_method") {
		return "An account already exists for that email. Sign in with your existing login method.";
	}

	if (error) {
		return "Could not complete OAuth sign in. Try again or use another login method.";
	}

	return undefined;
}
