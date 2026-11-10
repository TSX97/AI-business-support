import './App.css';
import PageFooter from './components/PageFooter';
import PageHeader from './components/PageHeader';
import RegistrationForm from './components/RegistrationForm';
import ChatPage from './components/chat/ChatPage';

function App() {
    if (window.location.pathname === '/chat') {
        return <ChatPage />;
    }

    return (
        <main className="page">
            <PageHeader />
            <section className="registration-wrapper">
                <RegistrationForm />
            </section>
            <PageFooter />
        </main>
    );
}

export default App;
