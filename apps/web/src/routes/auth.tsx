import { Logo } from '#/components/ui/logo'
import { createFileRoute, Link, Outlet, useLocation } from '@tanstack/react-router'

export const Route = createFileRoute('/auth')({
    component: AuthLayout,
})

function AuthLayout() {
    const location = useLocation()

    const bottomText =
        location.pathname === '/auth/signup' ?
            <p>already have an account? <Link to='/auth/login' search={{ oauth_error: undefined }} className='underline underline-offset-2' viewTransition>sign in</Link></p>
            : <p>do not have an account? <Link to='/auth/signup' className='underline underline-offset-2' viewTransition>sign up</Link></p>

    return <main className='min-h-screen bg-background text-foreground'>
        <div className='absolute inset-4 border border-border-soft flex z-10 tracking-tight justify-between'>

            <div className='flex-1 flex flex-col '>

                <div className='p-4 flex-1 w-md mx-auto  mt-12 '>
                    <Logo />
                    <Outlet />
                    <p className='text-center mt-6 text-sm text-muted-foreground [&_a]:text-foreground [&_a]:decoration-primary/70 [&_a]:transition-colors hover:[&_a]:text-primary'>{bottomText}</p>
                </div>
            </div>
        </div>
    </main>

}
