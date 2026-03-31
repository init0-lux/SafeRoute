import React, { useState, useEffect } from 'react';
import {
    View,
    Text,
    StyleSheet,
    TouchableOpacity,
    FlatList,
    TextInput,
    Alert,
    ActivityIndicator,
} from 'react-native';
import { router } from 'expo-router';
import { Ionicons } from '@expo/vector-icons';
import { SafeAreaView } from 'react-native-safe-area-context';
import { Colors, Spacing, BorderRadius, FontSizes } from '@/constants/theme';
import {
    TrustedContact,
    PendingTrustRequest,
    getTrustedContacts,
    getPendingRequests,
    createTrustedContactRequest,
    acceptTrustedContactRequest,
    rejectTrustedContactRequest,
    deleteTrustedContact,
} from '@/services/contacts';

type TabType = 'contacts' | 'pending';

export default function ContactsScreen() {
    const [activeTab, setActiveTab] = useState<TabType>('contacts');
    const [contacts, setContacts] = useState<TrustedContact[]>([]);
    const [pendingRequests, setPendingRequests] = useState<PendingTrustRequest[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [isAddMode, setIsAddMode] = useState(false);
    
    const [newName, setNewName] = useState('');
    const [newPhone, setNewPhone] = useState('');
    const [isSubmitting, setIsSubmitting] = useState(false);

    useEffect(() => {
        fetchData();
    }, []);

    const fetchData = async () => {
        setIsLoading(true);
        const [contactsData, pendingData] = await Promise.all([
            getTrustedContacts(),
            getPendingRequests(),
        ]);
        setContacts(contactsData);
        setPendingRequests(pendingData);
        setIsLoading(false);
    };

    const handleAddContact = async () => {
        const trimmedName = newName.trim();
        const trimmedPhone = newPhone.trim();

        if (!trimmedName || !trimmedPhone) {
            Alert.alert('Validation', 'Please fill out both name and phone number.');
            return;
        }

        // Phone must be digits only (after stripping spaces/dashes), at least 10 digits
        const digitsOnly = trimmedPhone.replace(/[\s\-\+\(\)]/g, '');
        if (digitsOnly.length < 10 || !/^\d+$/.test(digitsOnly)) {
            Alert.alert('Invalid Phone', 'Please enter a valid phone number with at least 10 digits.');
            return;
        }

        setIsSubmitting(true);
        try {
            await createTrustedContactRequest({
                name: trimmedName,
                phone: trimmedPhone,
            });
            
            Alert.alert(
                'Request Sent',
                `A trust request has been sent to ${trimmedName}. They will need to accept it to become your trusted contact.`
            );
            setNewName('');
            setNewPhone('');
            setIsAddMode(false);
            fetchData();
        } catch (err: any) {
            const msg = err?.message || 'Failed to send request';
            Alert.alert('Error', msg);
        } finally {
            setIsSubmitting(false);
        }
    };

    const handleAcceptRequest = async (request: PendingTrustRequest) => {
        if (!request.accept_token) {
            Alert.alert('Error', 'Missing accept token');
            return;
        }

        try {
            await acceptTrustedContactRequest(request.id, request.accept_token);
            Alert.alert('Accepted', `${request.requester_name || 'Contact'} is now your trusted contact!`);
            fetchData();
        } catch (err: any) {
            Alert.alert('Error', err?.message || 'Failed to accept request');
        }
    };

    const handleRejectRequest = async (request: PendingTrustRequest) => {
        Alert.alert(
            'Reject Request',
            `Are you sure you want to reject the request from ${request.requester_name || 'this user'}?`,
            [
                { text: 'Cancel', style: 'cancel' },
                {
                    text: 'Reject',
                    style: 'destructive',
                    onPress: async () => {
                        try {
                            await rejectTrustedContactRequest(request.id);
                            setPendingRequests((prev) => prev.filter((r) => r.id !== request.id));
                        } catch (err: any) {
                            Alert.alert('Error', err?.message || 'Failed to reject request');
                        }
                    },
                },
            ]
        );
    };

    const handleDelete = (contactId: string, name: string) => {
        Alert.alert(
            'Remove Contact',
            `Are you sure you want to remove ${name}? They will no longer receive SOS alerts.`,
            [
                { text: 'Cancel', style: 'cancel' },
                {
                    text: 'Remove',
                    style: 'destructive',
                    onPress: async () => {
                        try {
                            await deleteTrustedContact(contactId);
                            setContacts((prev) => prev.filter((c) => c.id !== contactId));
                        } catch (err: any) {
                            Alert.alert('Error', err?.message || 'Failed to remove contact');
                        }
                    },
                },
            ]
        );
    };

    const renderContactItem = ({ item }: { item: TrustedContact }) => (
        <View style={styles.contactCard}>
            <View style={styles.contactAvatar}>
                <Text style={styles.avatarText}>{item.name.charAt(0).toUpperCase()}</Text>
            </View>
            <View style={styles.contactInfo}>
                <Text style={styles.contactName}>{item.name}</Text>
                <Text style={styles.contactPhone}>{item.phone}</Text>
            </View>
            <TouchableOpacity
                style={styles.deleteButton}
                onPress={() => handleDelete(item.id, item.name)}
                activeOpacity={0.7}
            >
                <Ionicons name="trash-outline" size={20} color={Colors.red} />
            </TouchableOpacity>
        </View>
    );

    const renderPendingItem = ({ item }: { item: PendingTrustRequest }) => (
        <View style={styles.contactCard}>
            <View style={[styles.contactAvatar, { backgroundColor: Colors.orange }]}>
                <Text style={styles.avatarText}>
                    {(item.requester_name || item.name || '?').charAt(0).toUpperCase()}
                </Text>
            </View>
            <View style={styles.contactInfo}>
                <Text style={styles.contactName}>{item.requester_name || item.name}</Text>
                <Text style={styles.contactPhone}>{item.requester_phone || item.phone}</Text>
                <Text style={styles.pendingLabel}>Wants to add you as trusted contact</Text>
            </View>
            <View style={styles.pendingActions}>
                <TouchableOpacity
                    style={styles.acceptButton}
                    onPress={() => handleAcceptRequest(item)}
                    activeOpacity={0.7}
                >
                    <Ionicons name="checkmark" size={20} color={Colors.white} />
                </TouchableOpacity>
                <TouchableOpacity
                    style={styles.rejectButton}
                    onPress={() => handleRejectRequest(item)}
                    activeOpacity={0.7}
                >
                    <Ionicons name="close" size={20} color={Colors.white} />
                </TouchableOpacity>
            </View>
        </View>
    );

    return (
        <SafeAreaView style={styles.container}>
            {/* Header */}
            <View style={styles.header}>
                <TouchableOpacity onPress={() => router.back()} activeOpacity={0.7} style={styles.backButton}>
                    <Ionicons name="arrow-back" size={24} color={Colors.white} />
                </TouchableOpacity>
                <Text style={styles.headerTitle}>Trusted Contacts</Text>
                <TouchableOpacity onPress={() => setIsAddMode(!isAddMode)} activeOpacity={0.7}>
                    <Ionicons name={isAddMode ? "close" : "add"} size={28} color={Colors.purple} />
                </TouchableOpacity>
            </View>

            {/* Tabs */}
            <View style={styles.tabContainer}>
                <TouchableOpacity
                    style={[styles.tab, activeTab === 'contacts' && styles.tabActive]}
                    onPress={() => setActiveTab('contacts')}
                >
                    <Text style={[styles.tabText, activeTab === 'contacts' && styles.tabTextActive]}>
                        Contacts ({contacts.length})
                    </Text>
                </TouchableOpacity>
                <TouchableOpacity
                    style={[styles.tab, activeTab === 'pending' && styles.tabActive]}
                    onPress={() => setActiveTab('pending')}
                >
                    <Text style={[styles.tabText, activeTab === 'pending' && styles.tabTextActive]}>
                        Pending ({pendingRequests.length})
                    </Text>
                    {pendingRequests.length > 0 && (
                        <View style={styles.badge}>
                            <Text style={styles.badgeText}>{pendingRequests.length}</Text>
                        </View>
                    )}
                </TouchableOpacity>
            </View>

            {/* Add Contact Form (Conditional) */}
            {isAddMode && (
                <View style={styles.addForm}>
                    <Text style={styles.formTitle}>Send Trust Request</Text>
                    <Text style={styles.formSubtitle}>
                        The contact will receive a notification to accept your request.
                    </Text>
                    <TextInput
                        style={styles.input}
                        placeholder="Contact Name (e.g., Mom)"
                        placeholderTextColor={Colors.gray}
                        value={newName}
                        onChangeText={setNewName}
                    />
                    <TextInput
                        style={styles.input}
                        placeholder="Phone Number (+91...)"
                        placeholderTextColor={Colors.gray}
                        keyboardType="phone-pad"
                        value={newPhone}
                        onChangeText={setNewPhone}
                    />
                    <TouchableOpacity
                        style={[styles.submitButton, isSubmitting && { opacity: 0.6 }]}
                        onPress={handleAddContact}
                        disabled={isSubmitting}
                    >
                        {isSubmitting ? (
                            <ActivityIndicator size="small" color={Colors.bgDark} />
                        ) : (
                            <Text style={styles.submitButtonText}>Send Request</Text>
                        )}
                    </TouchableOpacity>
                </View>
            )}

            {/* Content */}
            {isLoading ? (
                <View style={styles.center}>
                    <ActivityIndicator size="large" color={Colors.purple} />
                </View>
            ) : activeTab === 'contacts' ? (
                contacts.length === 0 ? (
                    <View style={styles.center}>
                        <Ionicons name="people-outline" size={64} color={Colors.grayDark} />
                        <Text style={styles.emptyTitle}>No Trusted Contacts</Text>
                        <Text style={styles.emptySub}>
                            Add trusted contacts so they can be alerted instantly if you trigger an SOS.
                        </Text>
                    </View>
                ) : (
                    <FlatList
                        data={contacts}
                        keyExtractor={(item) => item.id}
                        renderItem={renderContactItem}
                        contentContainerStyle={styles.listContent}
                        showsVerticalScrollIndicator={false}
                    />
                )
            ) : (
                pendingRequests.length === 0 ? (
                    <View style={styles.center}>
                        <Ionicons name="mail-outline" size={64} color={Colors.grayDark} />
                        <Text style={styles.emptyTitle}>No Pending Requests</Text>
                        <Text style={styles.emptySub}>
                            When someone wants to add you as a trusted contact, it will appear here.
                        </Text>
                    </View>
                ) : (
                    <FlatList
                        data={pendingRequests}
                        keyExtractor={(item) => item.id}
                        renderItem={renderPendingItem}
                        contentContainerStyle={styles.listContent}
                        showsVerticalScrollIndicator={false}
                    />
                )
            )}
        </SafeAreaView>
    );
}

const styles = StyleSheet.create({
    container: {
        flex: 1,
        backgroundColor: Colors.bgDark,
    },
    header: {
        flexDirection: 'row',
        alignItems: 'center',
        justifyContent: 'space-between',
        paddingHorizontal: Spacing.lg,
        paddingTop: Spacing.md,
        paddingBottom: Spacing.lg,
    },
    backButton: {
        width: 40,
        height: 40,
        justifyContent: 'center',
    },
    headerTitle: {
        fontSize: FontSizes.lg,
        fontWeight: '700',
        color: Colors.white,
    },
    center: {
        flex: 1,
        justifyContent: 'center',
        alignItems: 'center',
        paddingHorizontal: Spacing.xl,
    },
    emptyTitle: {
        fontSize: FontSizes.md,
        fontWeight: '600',
        color: Colors.grayLight,
        marginTop: Spacing.lg,
        marginBottom: Spacing.sm,
    },
    emptySub: {
        fontSize: FontSizes.sm,
        color: Colors.gray,
        textAlign: 'center',
        lineHeight: 20,
    },
    listContent: {
        paddingHorizontal: Spacing.lg,
        paddingTop: Spacing.md,
        paddingBottom: 100,
    },
    contactCard: {
        flexDirection: 'row',
        alignItems: 'center',
        backgroundColor: Colors.cardBg,
        padding: Spacing.md,
        borderRadius: BorderRadius.md,
        marginBottom: Spacing.md,
        borderWidth: 1,
        borderColor: Colors.border,
    },
    contactAvatar: {
        width: 48,
        height: 48,
        borderRadius: 24,
        backgroundColor: Colors.purple,
        justifyContent: 'center',
        alignItems: 'center',
        marginRight: Spacing.md,
    },
    avatarText: {
        color: Colors.bgDark,
        fontSize: FontSizes.lg,
        fontWeight: '800',
    },
    contactInfo: {
        flex: 1,
    },
    contactName: {
        fontSize: FontSizes.md,
        fontWeight: '700',
        color: Colors.white,
        marginBottom: 2,
    },
    contactPhone: {
        fontSize: FontSizes.sm,
        color: Colors.grayLight,
    },
    deleteButton: {
        padding: Spacing.sm,
    },
    addForm: {
        backgroundColor: Colors.cardBg,
        marginHorizontal: Spacing.lg,
        marginBottom: Spacing.lg,
        padding: Spacing.lg,
        borderRadius: BorderRadius.md,
        borderWidth: 1,
        borderColor: Colors.border,
    },
    formTitle: {
        fontSize: FontSizes.md,
        fontWeight: '600',
        color: Colors.white,
        marginBottom: Spacing.sm,
    },
    formSubtitle: {
        fontSize: FontSizes.sm,
        color: Colors.gray,
        marginBottom: Spacing.lg,
    },
    input: {
        backgroundColor: Colors.bgDark,
        borderRadius: BorderRadius.sm,
        paddingHorizontal: Spacing.md,
        paddingVertical: Spacing.md,
        color: Colors.white,
        fontSize: FontSizes.md,
        marginBottom: Spacing.md,
        borderWidth: 1,
        borderColor: Colors.border,
    },
    submitButton: {
        backgroundColor: Colors.purple,
        borderRadius: BorderRadius.full,
        paddingVertical: Spacing.md,
        alignItems: 'center',
        marginTop: Spacing.sm,
    },
    submitButtonText: {
        color: Colors.bgDark,
        fontSize: FontSizes.md,
        fontWeight: '700',
    },
    tabContainer: {
        flexDirection: 'row',
        paddingHorizontal: Spacing.lg,
        marginBottom: Spacing.md,
        gap: Spacing.sm,
    },
    tab: {
        flex: 1,
        flexDirection: 'row',
        alignItems: 'center',
        justifyContent: 'center',
        paddingVertical: Spacing.md,
        borderRadius: BorderRadius.md,
        backgroundColor: Colors.cardBg,
        borderWidth: 1,
        borderColor: Colors.border,
    },
    tabActive: {
        backgroundColor: Colors.purple,
        borderColor: Colors.purple,
    },
    tabText: {
        fontSize: FontSizes.sm,
        fontWeight: '600',
        color: Colors.grayLight,
    },
    tabTextActive: {
        color: Colors.bgDark,
    },
    badge: {
        backgroundColor: Colors.sosRed,
        borderRadius: 10,
        paddingHorizontal: 6,
        paddingVertical: 2,
        marginLeft: 6,
    },
    badgeText: {
        fontSize: 10,
        fontWeight: '700',
        color: Colors.white,
    },
    pendingLabel: {
        fontSize: FontSizes.xs,
        color: Colors.orange,
        marginTop: 4,
    },
    pendingActions: {
        flexDirection: 'row',
        gap: Spacing.sm,
    },
    acceptButton: {
        width: 36,
        height: 36,
        borderRadius: 18,
        backgroundColor: Colors.success,
        justifyContent: 'center',
        alignItems: 'center',
    },
    rejectButton: {
        width: 36,
        height: 36,
        borderRadius: 18,
        backgroundColor: Colors.red,
        justifyContent: 'center',
        alignItems: 'center',
    },
});
