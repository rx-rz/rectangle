import { createFileRoute, Link, Outlet, useLocation } from '@tanstack/react-router'
import { Dot } from 'lucide-react'

export const Route = createFileRoute('/auth')({
    component: AuthLayout,
})

function AuthLayout() {
    const location = useLocation()

    const bottomText =
        location.pathname === '/auth/signup' ?
            <p>already have an account? <Link to='/auth/login' className='underline underline-offset-2'>sign in</Link></p>
            : <p>do not have an account? <Link to='/auth/signup' className='underline underline-offset-2'>sign up</Link></p>

    return <main className='min-h-screen'>
        <div className='absolute inset-4 border flex z-10 tracking-tighter justify-between'>
            <div className='flex-1 border-r relative bg-transparent '>
                <div className='flex flex-col h-full p-4'>
                    <img src="/cloud-dither.png" className='absolute inset-0 h-full object-cover opacity-40 -z-10' alt="" />
                    <img src="/logo.png" alt="" className='size-18 absolute -top-3 left-0' />
                    <div className='font-mono mt-auto flex flex-col text-sm opacity-80 animate-pulse gap-1'>
                        <p className='text-primary'>//</p>
                        <div>
                            <p>deployment infrastructure</p>
                            <p>built for scale</p>
                        </div>

                    </div>
                </div>
            </div>
            <div className='flex-1 font-mono flex flex-col'>
                <div className=' flex justify-between p-3 px-8 items-center  border-b'>
                    <p className='text-sm'>create account</p>
                    <Dot size={20} className='animate-pulse text-primary' />
                </div>
                <div className='p-4 flex-1 px-20 mt-12'>
                    <Outlet />
                </div>
                <div className='flex justify-between p-3 px-8 items-center text-sm  border-t'>
                    {bottomText}
                    <Dot size={20} className='animate-pulse text-primary' />
                </div>
            </div>
        </div>
    </main>

}
