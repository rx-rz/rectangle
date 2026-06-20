import { OauthContainer } from '#/features/auth/signup/containers/oauth'
import { SignupForm } from '#/features/auth/signup/containers/signup'
import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/auth/signup')({
    component: SignupRoute,
})

function SignupRoute() {
    return <div className='flex flex-col gap-6 w-full'>
        <SignupForm />
        <div className='flex items-center gap-4 text-sm'>
            <div className='bg-muted h-px flex-1'></div>
            OR
            <div className='bg-muted h-px flex-1 '></div>
            </div>
        <OauthContainer/>
    </div>
}
