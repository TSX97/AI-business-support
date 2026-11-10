import './App.css';
import PageFooter from './components/PageFooter';
import PageHeader from './components/PageHeader';
import RegistrationForm from './components/RegistrationForm';

function App() {
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
