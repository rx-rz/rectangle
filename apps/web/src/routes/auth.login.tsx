import { createFileRoute } from "@tanstack/react-router";
import { LoginForm } from "#/features/auth/login/containers/login";
import { OauthContainer } from "#/features/auth/signup/containers/oauth";

export const Route = createFileRoute("/auth/login")({
	component: RouteComponent,
});

function RouteComponent() {
	return <div className='flex flex-col gap-6'>
		<LoginForm />
		<div className='flex items-center gap-4'>
			<div className='bg-muted h-px flex-1'></div>
			OR
			<div className='bg-muted h-px flex-1 '></div>
		</div>
		<OauthContainer />
	</div>
}
